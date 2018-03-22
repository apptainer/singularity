/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/


#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <stdarg.h>
#include <unistd.h>
#include <errno.h>
#include <ctype.h>
#include <string.h>
#include <fcntl.h>
#include <poll.h>
#include <grp.h>
#include <link.h>
#include <sys/fsuid.h>
#include <sys/mount.h>
#include <sys/wait.h>
#include <sys/prctl.h>
#include <sys/socket.h>
#include <sys/stat.h>
#include <signal.h>
#include <sched.h>
#include <sys/socket.h>
#include <setjmp.h>
#include <sys/syscall.h>
#include <dlfcn.h>

#include "include/wrapper.h"
#include "include/message.h"

// from build directory
#include "librpc.h"

#define CLONE_STACK_SIZE    1024*1024
#define MAX_JSON_SIZE       64*1024
#define BUFSIZE             512

extern char **environ;

typedef struct fork_state_s {
    sigjmp_buf env;
} fork_state_t;

int setns(int fd, int nstype) {
    return syscall(__NR_setns, fd, nstype);
}

static const char *int2str(int num) {
    char *str = (char *)malloc(16);
    memset(str, 0, 16);
    snprintf(str, 15, "%d", num);
    str[15] = '\0';
    return(str);
}

/* copy paste from singularity code */
static int clone_fn(void *data_ptr) {
    fork_state_t *state = (fork_state_t *)data_ptr;
    siglongjmp(state->env, 1);
}

static int fork_ns(unsigned int flags) {
    fork_state_t state;

    if ( sigsetjmp(state.env, 1) ) {
        return 0;
    }

    int stack_size = CLONE_STACK_SIZE;
    void *child_stack_ptr = malloc(stack_size);
    if ( child_stack_ptr == 0 ) {
        errno = ENOMEM;
        return -1;
    }
    child_stack_ptr += stack_size;

    int retval = clone(clone_fn,
          child_stack_ptr,
          (SIGCHLD|flags),
          &state
         );
    return retval;
}

static void priv_escalate(void) {
    /*
     * We set effective uid/gid here to protect /proc/<pid> from being accessed
     * by users
     */
    print(VERBOSE, "Get root privileges");
    if ( seteuid(0) < 0 || setegid(0) < 0 ) {
        pfatal("Failed to set effective UID/GID to 0");
    }
}

static void enter_namespace(pid_t pid, int nstype) {
    int ns_fd;
    char buffer[256];
    char *namespace = NULL;

    switch(nstype) {
    case CLONE_NEWPID:
        namespace = strdup("pid");
        break;
    case CLONE_NEWNET:
        namespace = strdup("net");
        break;
    case CLONE_NEWIPC:
        namespace = strdup("ipc");
        break;
    case CLONE_NEWNS:
        namespace = strdup("mnt");
        break;
    case CLONE_NEWUTS:
        namespace = strdup("uts");
        break;
    case CLONE_NEWUSER:
        namespace = strdup("user");
        break;
#ifdef CLONE_NEWCGROUP
    case CLONE_NEWCGROUP:
        namespace = strdup("cgroup");
        break;
#endif
    default:
        pfatal("No namespace type specified");
    }

    memset(buffer, 0, 256);
    snprintf(buffer, 255, "/proc/%d/ns/%s", pid, namespace);

    print(DEBUG, "Opening namespace file descriptor %s", buffer);
    ns_fd = open(buffer, O_RDONLY);
    if ( ns_fd < 0 ) {
        pfatal("Failed to enter in namespace %s of PID %d: %s", namespace, pid, strerror(errno));
    }

    print(VERBOSE, "Entering in %s namespace", namespace);

    if ( setns(ns_fd, nstype) < 0 ) {
        pfatal("Failed to enter in namespace %s of PID %d: %s", namespace, pid, strerror(errno));
    }

    close(ns_fd);
    free(namespace);
}

static void setup_userns(const struct uidMapping *uidMapping, const struct gidMapping *gidMapping) {
    FILE *map_fp;
    int i;
    struct uidMapping *uidmap;
    struct gidMapping *gidmap;

    print(VERBOSE, "Create user namespace");

    if ( unshare(CLONE_NEWUSER) < 0 ) {
        pfatal("Failed to create user namespace");
    }

    print(DEBUG, "Write deny to set group file");
    map_fp = fopen("/proc/self/setgroups", "w+"); // Flawfinder: ignore
    if ( map_fp != NULL ) {
        fprintf(map_fp, "deny\n");
        if ( fclose(map_fp) < 0 ) {
            pfatal("Failed to write deny to setgroup file: %s\n", strerror(errno));
        }
    } else {
        pfatal("Could not write info to setgroups: %s\n", strerror(errno));
    }

    print(DEBUG, "Write to GID map");
    for ( i = 0; i < MAX_ID_MAPPING; i++ ) {
        gidmap = (struct gidMapping *)&gidMapping[i];
        if ( gidmap->size == 0 ) {
            break;
        }
        map_fp = fopen("/proc/self/gid_map", "w+"); // Flawfinder: ignore
        if ( map_fp != NULL ) {
            print(DEBUG, "Write line '%i %i %i' to gid_map", gidmap->containerID, gidmap->hostID, gidmap->size);
            fprintf(map_fp, "%i %i %i\n", gidmap->containerID, gidmap->hostID, gidmap->size);
            if ( fclose(map_fp) < 0 ) {
                pfatal("Failed to write to GID map: %s\n", strerror(errno));
            }
        } else {
            pfatal("Could not write parent info to gid_map: %s\n", strerror(errno));
        }
    }

    print(DEBUG, "Write to UID map");
    for ( i = 0; i < MAX_ID_MAPPING; i++ ) {
        uidmap = (struct uidMapping *)&uidMapping[i];
        if ( uidmap->size == 0 ) {
            break;
        }
        map_fp = fopen("/proc/self/uid_map", "w+"); // Flawfinder: ignore
        if ( map_fp != NULL ) {
            fprintf(map_fp, "%i %i %i\n", uidmap->containerID, uidmap->hostID, uidmap->size);
            if ( fclose(map_fp) < 0 ) {
                pfatal("Failed to write to UID map: %s\n", strerror(errno));
            }
        } else {
            pfatal("Could not write parent info to uid_map: %s\n", strerror(errno));
        }
    }
}

static unsigned char is_suid(void) {
    ElfW(auxv_t) *auxv;
    unsigned char suid = 0;
    char *progname = NULL;
    char *buffer = (char *)malloc(4096);
    int proc_auxv = open("/proc/self/auxv", O_RDONLY);

    print(VERBOSE, "Check if we are running as setuid");

    if ( proc_auxv < 0 ) {
        pfatal("Cant' open /proc/self/auxv: %s", strerror(errno));
    }

    /* use auxiliary vectors to determine if running privileged */
    memset(buffer, 0, 4096);
    if ( read(proc_auxv, buffer, 4092) < 0 ) {
        pfatal("Can't read auxiliary vectors: %s", strerror(errno));
    }

    auxv = (ElfW(auxv_t) *)buffer;

    for (; auxv->a_type != AT_NULL; auxv++) {
        if ( auxv->a_type == AT_SECURE ) {
            suid = (int)auxv->a_un.a_val;
            break;
        }
    }

    free(buffer);
    close(proc_auxv);

    return suid;
}

static void set_parent_death_signal(int signo) {
    print(DEBUG, "Set parent death signal to %d", signo);
    if ( prctl(PR_SET_PDEATHSIG, signo) < 0 ) {
        pfatal("Failed to set parent death signal");
    }
}

void do_nothing(int sig) {
    return;
}

int main(int argc, char **argv) {
    char *json_stdin;
    char *env[8] = {0};
    int stage_socket[2];
    pid_t stage1, stage2;
    uid_t uid = getuid();
    gid_t gid = getgid();
    struct cConfig config;
    sigset_t mask;
    char *loglevel;
    char *runtime;

    loglevel = getenv("MESSAGELEVEL");
    if ( loglevel != NULL ) {
        loglevel = strdup(loglevel);
    } else {
        pfatal("MESSAGELEVEL environment variable isn't set");
    }

    runtime = getenv("SRUNTIME");
    if ( runtime != NULL ) {
        runtime = strdup(runtime);
    } else {
        pfatal("SRUNTIME environment variable isn't set");
    }

    print(VERBOSE, "Container runtime");

    memset(&config, 0, sizeof(config));

    config.isSuid = is_suid();

    if ( config.isSuid ) {
        print(DEBUG, "Drop privileges");
        if ( setegid(gid) < 0 || seteuid(uid) < 0 ) {
            pfatal("Failed to drop privileges");
        }
    }

    /* reset environment variables */
    environ = env;

    if ( loglevel != NULL ) {
        setenv("MESSAGELEVEL", loglevel, 1);
        free(loglevel);
    }

    if ( runtime != NULL ) {
        setenv("SRUNTIME", runtime, 1);
        free(runtime);
    }

    print(DEBUG, "Check PR_SET_NO_NEW_PRIVS support");
#ifdef PR_SET_NO_NEW_PRIVS
    print(DEBUG, "PR_SET_NO_NEW_PRIVS supported");
    config.hasNoNewPrivs = 1;
#else
    print(DEBUG, "PR_SET_NO_NEW_PRIVS not supported");
    config.hasNoNewPrivs = 0;
#endif

    /* read json configuration from stdin */
    print(DEBUG, "Read json configuration from stdin");
    int std = open("/proc/self/fd/1", O_RDONLY);

    json_stdin = (char *)malloc(MAX_JSON_SIZE);
    if ( json_stdin == NULL ) {
        pfatal("Memory allocation failure");
    }

    memset(json_stdin, 0, MAX_JSON_SIZE);
    if ( ( config.jsonConfSize = read(STDIN_FILENO, json_stdin, MAX_JSON_SIZE - 1) ) <= 0 ) {
        pfatal("Read from stdin failed");
    }

    /* back to terminal stdin */
    if ( isatty(std) ) {
        print(DEBUG, "Run in terminal, restore stdin");
        dup2(std, 0);
    }
    close(std);

    print(DEBUG, "Set SIGCHLD signal handler");
    signal(SIGCHLD, &do_nothing);

    /* for security reasons use socketpair only for process communications */
    if ( socketpair(AF_UNIX, SOCK_DGRAM, 0, stage_socket) < 0 ) {
        pfatal("Failed to create communication socket");
    }

    stage1 = fork();
    if ( stage1 == 0 ) {
        setenv("SCONTAINER_STAGE", "1", 1);
        setenv("SCONTAINER_SOCKET", int2str(stage_socket[1]), 1);

        print(VERBOSE, "Spawn scontainer stage 1");

        close(stage_socket[0]);

        /*
         *  stage1 is responsible for singularity configuration file parsing, handle user input,
         *  read capabilities, check what namespaces is required.
         */
        if ( config.isSuid ) {
            priv_escalate();
        }

        print(VERBOSE, "Execute scontainer stage 1");

        execle("/tmp/scontainer", "/tmp/scontainer", NULL, environ);
        pfatal("Scontainer stage 1 execution failed");
    } else if ( stage1 > 0 ) {
        pid_t parent = getpid();
        int status;
        struct pollfd fds;
        void *readbuf = &config;
        size_t readsize = sizeof(config);

        close(stage_socket[1]);

        fds.fd = stage_socket[0];
        fds.events = POLLIN;
        fds.revents = 0;

        print(DEBUG, "Send C runtime configuration to scontainer stage 1");

        /* send runtime configuration to scontainer (CGO) */
        if ( write(stage_socket[0], &config, sizeof(config)) != sizeof(config) ) {
            pfatal("Failed to send runtime configuration");
        }

        print(DEBUG, "Send JSON runtime configuration to scontainer stage 1");

        /* send json configuration to scontainer */
        if ( write(stage_socket[0], json_stdin, config.jsonConfSize) != config.jsonConfSize ) {
            pfatal("Copy json configuration failed");
        }

        print(DEBUG, "Wait C and JSON runtime configuration from scontainer stage 1");

        while ( poll(&fds, 1, -1) >= 0 ) {
            if ( fds.revents == POLLIN ) {
                int ret;

                print(DEBUG, "Receiving configuration from scontainer stage 1");
                if ( (ret = read(stage_socket[0], readbuf, readsize)) != readsize ) {
                    pfatal("Failed to read communication pipe %d", ret);
                }
                if ( readbuf == json_stdin ) {
                    break;
                }
                readbuf = json_stdin;
                readsize = config.jsonConfSize;
                if ( config.jsonConfSize >= MAX_JSON_SIZE) {
                    pfatal("json configuration too big");
                }
                json_stdin[config.jsonConfSize] = '\0';
            }
        }

        print(DEBUG, "Wait completion of scontainer stage1");
        if ( wait(&status) != stage1 ) {
            pfatal("Can't wait child");
        }

        if ( WIFEXITED(status) || WIFSIGNALED(status) ) {
            if ( WEXITSTATUS(status) != 0 ) {
                pfatal("Child exit with status %d", WEXITSTATUS(status));
            }
        }
        close(stage_socket[0]);

        /* block SIGCHLD signal handled later by scontainer/smaster */
        print(DEBUG, "Set child signal mask");
        sigemptyset(&mask);
        sigaddset(&mask, SIGCHLD);
        if (sigprocmask(SIG_SETMASK, &mask, NULL) == -1) {
            pfatal("Blocked signals error");
        }

        if ( config.isInstance ) {
            print(VERBOSE, "Run as instance");
            int forked = fork();
            if ( forked == 0 ) {
                int i;
                if ( chdir("/") < 0 ) {
                    pfatal("Can't change directory to /");
                }
                if ( setsid() < 0 ) {
                    pfatal("Can't set session leader");
                }
                umask(0);

                print(DEBUG, "Close all file descriptor");
                for( i = sysconf(_SC_OPEN_MAX); i > 2; i-- ) {
                    close(i);
                }
            } else {
                int status;

                print(DEBUG, "Wait child process signaling SIGSTOP");
                waitpid(forked, &status, WUNTRACED);
                if ( WIFSTOPPED(status) ) {
                    print(DEBUG, "Send SIGCONT to child process");
                    kill(forked, SIGCONT);
                    return(0);
                }
                if ( WIFEXITED(status) || WIFSIGNALED(status) ) {
                    print(VERBOSE, "Child process exited with status %d", WEXITSTATUS(status));
                    return(WEXITSTATUS(status));
                }
                return(-1);
            }
        }

        if ( (config.nsFlags & CLONE_NEWUSER) == 0 ) {
            priv_escalate();
        } else {
            if ( config.userPid ) {
                enter_namespace(config.userPid, CLONE_NEWUSER);
            } else {
                setup_userns(&config.uidMapping[0], &config.gidMapping[0]);
            }
        }

        print(DEBUG, "Create socketpair communication between smaster and scontainer");
        if ( socketpair(AF_UNIX, SOCK_STREAM, 0, stage_socket) < 0 ) {
            pfatal("Failed to create communication socket");
        }

        /* enforce PID namespace if NO_NEW_PRIVS not supported  */
        if ( config.hasNoNewPrivs == 0 ) {
            print(VERBOSE, "No PR_SET_NO_NEW_PRIVS support, enforcing PID namespace");
            config.nsFlags |= CLONE_NEWPID;
        }

        if ( config.pidPid ) {
            enter_namespace(config.pidPid, CLONE_NEWPID);
            stage2 = fork();
        } else {
            if ( config.nsFlags & CLONE_NEWPID ) {
                print(VERBOSE, "Create pid namespace");
                stage2 = fork_ns(CLONE_NEWPID);
            } else {
                stage2 = fork();
            }
        }

        if ( stage2 == 0 ) {
            /* at this stage we are PID 1 if PID namespace requested */
            unsigned char notification = 'S';
            int rpc_socket[2];
            pid_t child;

            print(VERBOSE, "Spawn scontainer stage 2");

            set_parent_death_signal(SIGKILL);

            if ( config.netPid ) {
                enter_namespace(config.netPid, CLONE_NEWNET);
            } else {
                if ( config.nsFlags & CLONE_NEWNET ) {
                    print(VERBOSE, "Create net namespace");
                    if ( unshare(CLONE_NEWNET) < 0 ) {
                        pfatal("failed to create network namespace");
                    }
                }
            }
            if ( config.utsPid ) {
                enter_namespace(config.utsPid, CLONE_NEWUTS);
            } else {
                if ( config.nsFlags & CLONE_NEWUTS ) {
                    print(VERBOSE, "Create uts namespace");
                    if ( unshare(CLONE_NEWUTS) < 0 ) {
                        pfatal("failed to create uts namespace");
                    }
                }
            }
            if ( config.ipcPid ) {
                enter_namespace(config.ipcPid, CLONE_NEWIPC);
            } else {
                if ( config.nsFlags & CLONE_NEWIPC ) {
                    print(VERBOSE, "Create ipc namespace");
                    if ( unshare(CLONE_NEWIPC) < 0 ) {
                        pfatal("failed to create ipc namespace");
                    }
                }
            }
#ifdef CLONE_NEWCGROUP
            if ( config.cgroupPid ) {
                enter_namespace(config.cgroupPid, CLONE_NEWCGROUP);
            } else {
                if ( config.nsFlags & CLONE_NEWCGROUP ) {
                    print(VERBOSE, "Create cgroup namespace");
                    if ( unshare(CLONE_NEWCGROUP) < 0 ) {
                        pfatal("failed to create cgroup namespace");
                    }
                }
            }
#endif
            if ( config.mntPid ) {
                enter_namespace(config.mntPid, CLONE_NEWNS);
            } else {
                print(VERBOSE, "Unshare filesystem and create mount namespace");
                if ( unshare(CLONE_FS) < 0 ) {
                    pfatal("Failed to unshare filesystem");
                }
                if ( unshare(CLONE_NEWNS) < 0 ) {
                    pfatal("Failed to unshare mount namespace");
                }
            }

            print(DEBUG, "Create RPC socketpair for communication between scontainer and RPC server");
            if ( socketpair(AF_UNIX, SOCK_STREAM, 0, rpc_socket) < 0 ) {
                pfatal("Failed to create communication socket");
            }

            close(stage_socket[0]);

            if ( write(stage_socket[1], &notification, 1) != 1 ) {
                pfatal("failed to send start notification to parent process");
            }

            child = fork();
            if ( child == 0 ) {
                void *handle;
                GoInt (*rpcserver)(GoInt socket);

                print(VERBOSE, "Spawn RPC server");

                close(stage_socket[1]);
                close(rpc_socket[0]);

                /* return to host network namespace for network setup */
                print(DEBUG, "Return to host network namespace");
                if ( config.nsFlags & CLONE_NEWNET && (config.nsFlags & CLONE_NEWUSER) == 0 ) {
                    enter_namespace(parent, CLONE_NEWNET);
                }

                /* Use setfsuid to address issue about root_squash filesystems option */
                if ( config.isSuid ) {
                    if ( setfsuid(uid) < 0 ) {
                        pfatal("Failed to set fs uid");
                    }
                }

                /*
                 * If we execute rpc server there, we will lose all capabilities during execve
                 * when using user namespace, so we won't be able to serve any privileged operations,
                 * a solution is to load rpc server as a shared library
                 */
                print(DEBUG, "Load librpc.so");
                handle = dlopen("/tmp/librpc.so", RTLD_LAZY);
                if ( handle == NULL ) {
                    pfatal("Failed to load shared lib librpc.so");
                }
                rpcserver = (GoInt (*)(GoInt))dlsym(handle, "RPCServer");
                if ( rpcserver == NULL ) {
                    pfatal("Failed to find symbol");
                }

                free(json_stdin);

                print(VERBOSE, "Serve RPC requests");

                return(rpcserver((GoInt)rpc_socket[1]));
            } else if ( child > 0 ) {
                setenv("SCONTAINER_STAGE", "2", 1);
                setenv("SCONTAINER_SOCKET", int2str(stage_socket[1]), 1);
                setenv("SCONTAINER_RPC_SOCKET", int2str(rpc_socket[0]), 1);

                close(rpc_socket[1]);

                /* send json configuration to smaster */
                print(DEBUG, "Send JSON configuration to smaster");
                if ( write(stage_socket[1], json_stdin, config.jsonConfSize) != config.jsonConfSize ) {
                    pfatal("copy json configuration failed");
                }

                print(VERBOSE, "Execute scontainer stage 2");
                execle("/tmp/scontainer", "/tmp/scontainer", NULL, environ);
            }
            pfatal("Failed to execute container");
        } else if ( stage2 > 0 ) {
            unsigned char notification;

            setenv("SMASTER_INSTANCE", int2str(config.isInstance), 1);
            setenv("SMASTER_CONTAINER_PID", int2str(stage2), 1);
            setenv("SMASTER_SOCKET", int2str(stage_socket[0]), 1);

            config.containerPid = stage2;

            print(VERBOSE, "Spawn smaster process");

            close(stage_socket[1]);

            /* wait start notification from child */
            if ( read(stage_socket[0], &notification, 1) != 1 ) {
                pfatal("failed to get start notification from child process");
            }

            /* send runtime configuration to scontainer (CGO) */
            print(DEBUG, "Send C runtime configuration to scontainer stage 2");
            if ( write(stage_socket[0], &config, sizeof(config)) != sizeof(config) ) {
                pfatal("failed to send runtime configuration");
            }

            /* send json configuration to scontainer */
            print(DEBUG, "Send JSON runtime configuration to scontainer stage 2");
            if ( write(stage_socket[0], json_stdin, config.jsonConfSize) != config.jsonConfSize ) {
                pfatal("copy json configuration failed");
            }

            print(VERBOSE, "Execute smaster process");
            execle("/tmp/smaster", "/tmp/smaster", NULL, environ);
        }
        pfatal("Failed to create container namespaces");
    }
    return(0);
}
