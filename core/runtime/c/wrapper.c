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

#include "runtime/c/include/wrapper.h"
#include "lib/util/message.h"

// from build directory
#include "buildtree/librpc.h"

#define CLONE_STACK_SIZE    1024*1024
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
    char *child_stack_ptr = malloc(stack_size);
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
    singularity_message(VERBOSE, "Get root privileges\n");
    if ( seteuid(0) < 0 || setegid(0) < 0 ) {
        singularity_message(ERROR, "Failed to set effective UID/GID to 0\n");
        exit(1);
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
        singularity_message(ERROR, "No namespace type specified\n");
        exit(1);
    }

    memset(buffer, 0, 256);
    snprintf(buffer, 255, "/proc/%d/ns/%s", pid, namespace);

    singularity_message(DEBUG, "Opening namespace file descriptor %s\n", buffer);
    ns_fd = open(buffer, O_RDONLY);
    if ( ns_fd < 0 ) {
        singularity_message(ERROR, "Failed to enter in namespace %s of PID %d: %s\n", namespace, pid, strerror(errno));
        exit(1);
    }

    singularity_message(VERBOSE, "Entering in %s namespace\n", namespace);

    if ( setns(ns_fd, nstype) < 0 ) {
        singularity_message(ERROR, "Failed to enter in namespace %s of PID %d: %s\n", namespace, pid, strerror(errno));
        exit(1);
    }

    close(ns_fd);
    free(namespace);
}

static void setup_userns(const struct uidMapping *uidMapping, const struct gidMapping *gidMapping) {
    FILE *map_fp;
    int i;
    struct uidMapping *uidmap;
    struct gidMapping *gidmap;

    singularity_message(VERBOSE, "Create user namespace\n");

    if ( unshare(CLONE_NEWUSER) < 0 ) {
        singularity_message(ERROR, "Failed to create user namespace\n");
        exit(1);
    }

    singularity_message(DEBUG, "Write deny to set group file\n");
    map_fp = fopen("/proc/self/setgroups", "w+"); // Flawfinder: ignore
    if ( map_fp != NULL ) {
        fprintf(map_fp, "deny\n");
        if ( fclose(map_fp) < 0 ) {
            singularity_message(ERROR, "Failed to write deny to setgroup file: %s\n", strerror(errno));
            exit(1);
        }
    } else {
        singularity_message(ERROR, "Could not write info to setgroups: %s\n", strerror(errno));
        exit(1);
    }

    singularity_message(DEBUG, "Write to GID map\n");
    for ( i = 0; i < MAX_ID_MAPPING; i++ ) {
        gidmap = (struct gidMapping *)&gidMapping[i];
        if ( gidmap->size == 0 ) {
            break;
        }
        map_fp = fopen("/proc/self/gid_map", "w+"); // Flawfinder: ignore
        if ( map_fp != NULL ) {
            singularity_message(DEBUG, "Write line '%i %i %i' to gid_map\n", gidmap->containerID, gidmap->hostID, gidmap->size);
            fprintf(map_fp, "%i %i %i\n", gidmap->containerID, gidmap->hostID, gidmap->size);
            if ( fclose(map_fp) < 0 ) {
                singularity_message(ERROR, "Failed to write to GID map: %s\n", strerror(errno));
                exit(1);
            }
        } else {
            singularity_message(ERROR, "Could not write parent info to gid_map: %s\n", strerror(errno));
            exit(1);
        }
    }

    singularity_message(DEBUG, "Write to UID map\n");
    for ( i = 0; i < MAX_ID_MAPPING; i++ ) {
        uidmap = (struct uidMapping *)&uidMapping[i];
        if ( uidmap->size == 0 ) {
            break;
        }
        map_fp = fopen("/proc/self/uid_map", "w+"); // Flawfinder: ignore
        if ( map_fp != NULL ) {
            fprintf(map_fp, "%i %i %i\n", uidmap->containerID, uidmap->hostID, uidmap->size);
            if ( fclose(map_fp) < 0 ) {
                singularity_message(ERROR, "Failed to write to UID map: %s\n", strerror(errno));
                exit(1);
            }
        } else {
            singularity_message(ERROR, "Could not write parent info to uid_map: %s\n", strerror(errno));
            exit(1);
        }
    }
}

static unsigned char is_suid(void) {
    ElfW(auxv_t) *auxv;
    unsigned char suid = 0;
    char *buffer = (char *)malloc(4096);
    int proc_auxv = open("/proc/self/auxv", O_RDONLY);

    singularity_message(VERBOSE, "Check if we are running as setuid\n");

    if ( proc_auxv < 0 ) {
        singularity_message(ERROR, "Cant' open /proc/self/auxv: %s\n", strerror(errno));
        exit(1);
    }

    /* use auxiliary vectors to determine if running privileged */
    memset(buffer, 0, 4096);
    if ( read(proc_auxv, buffer, 4092) < 0 ) {
        singularity_message(ERROR, "Can't read auxiliary vectors: %s\n", strerror(errno));
        exit(1);
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
    singularity_message(DEBUG, "Set parent death signal to %d\n", signo);
    if ( prctl(PR_SET_PDEATHSIG, signo) < 0 ) {
        singularity_message(ERROR, "Failed to set parent death signal\n");
        exit(1);
    }
}

void do_nothing(int sig) {
    (void)sig;
    return;
}

int main(int argc, char **argv) {
    (void)argc;
    (void)argv;
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
    int output[2];
    int input[2];

    loglevel = getenv("SINGULARITY_MESSAGELEVEL");
    if ( loglevel != NULL ) {
        loglevel = strdup(loglevel);
    } else {
        singularity_message(ERROR, "SINGULARITY_MESSAGELEVEL environment variable isn't set\n");
        exit(1);
    }

    runtime = getenv("SRUNTIME");
    if ( runtime != NULL ) {
        runtime = strdup(runtime);
    } else {
        singularity_message(ERROR, "SRUNTIME environment variable isn't set\n");
        exit(1);
    }

    singularity_message(VERBOSE, "Container runtime\n");

    memset(&config, 0, sizeof(config));

    config.isSuid = is_suid();

    if ( config.isSuid ) {
        singularity_message(DEBUG, "Drop privileges\n");
        if ( setegid(gid) < 0 || seteuid(uid) < 0 ) {
            singularity_message(ERROR, "Failed to drop privileges\n");
            exit(1);
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

    singularity_message(DEBUG, "Check PR_SET_NO_NEW_PRIVS support\n");
#ifdef PR_SET_NO_NEW_PRIVS
    singularity_message(DEBUG, "PR_SET_NO_NEW_PRIVS supported\n");
    config.hasNoNewPrivs = 1;
#else
    singularity_message(DEBUG, "PR_SET_NO_NEW_PRIVS not supported\n");
    config.hasNoNewPrivs = 0;
#endif

    /* read json configuration from stdin */
    singularity_message(DEBUG, "Read json configuration from stdin\n");
    int std = open("/proc/self/fd/1", O_RDONLY);

    json_stdin = (char *)malloc(MAX_JSON_SIZE);
    if ( json_stdin == NULL ) {
        singularity_message(ERROR, "Memory allocation failure\n");
        exit(1);
    }

    memset(json_stdin, 0, MAX_JSON_SIZE);
    if ( ( config.jsonConfSize = read(STDIN_FILENO, json_stdin, MAX_JSON_SIZE - 1) ) <= 0 ) {
        singularity_message(ERROR, "Read from stdin failed\n");
        exit(1);
    }

    /* back to terminal stdin */
    if ( isatty(std) ) {
        singularity_message(DEBUG, "Run in terminal, restore stdin\n");
        dup2(std, STDIN_FILENO);
    }
    close(std);

    singularity_message(DEBUG, "Set SIGCHLD signal handler\n");
    signal(SIGCHLD, &do_nothing);

    if ( pipe2(output, 0) < 0 ) {
        singularity_message(ERROR, "failed to create output process pipes\n");
        exit(1);
    }
    if ( pipe2(input, 0) < 0 ) {
        singularity_message(ERROR, "failed to create input process pipes\n");
        exit(1);
    }

    stage1 = fork();
    if ( stage1 == 0 ) {
        setenv("SCONTAINER_STAGE", "1", 1);

        close(output[0]);
        close(input[1]);

        if ( dup2(input[0], JOKER) < 0 ) {
            singularity_message(ERROR, "failed to create stdin pipe\n");
            exit(1);
        }
        close(input[0]);
        if ( dup2(output[1], STDOUT_FILENO) < 0 ) {
            singularity_message(ERROR, "failed to create stdout pipe\n");
            exit(1);
        }
        close(output[1]);

        singularity_message(VERBOSE, "Spawn scontainer stage 1\n");

        /*
         *  stage1 is responsible for singularity configuration file parsing, handle user input,
         *  read capabilities, check what namespaces is required.
         */
        if ( config.isSuid ) {
            priv_escalate();
        }

        singularity_message(VERBOSE, "Execute scontainer stage 1\n");

        singularity_message(VERBOSE, BUILDDIR "/scontainer\n");
        execle(BUILDDIR "/scontainer", BUILDDIR "/scontainer", NULL, environ);
        singularity_message(ERROR, "Scontainer stage 1 execution failed\n");
        exit(1);
    } else if ( stage1 > 0 ) {
        pid_t parent = getpid();
        int status;
        struct pollfd fds;

        close(output[1]);
        close(input[0]);

        fds.fd = output[0];
        fds.events = POLLIN;
        fds.revents = 0;

        singularity_message(DEBUG, "Send C runtime configuration to scontainer stage 1\n");

        /* send runtime configuration to scontainer (CGO) */
        if ( write(input[1], &config, sizeof(config)) != sizeof(config) ) {
            singularity_message(ERROR, "Failed to send runtime configuration\n");
            exit(1);
        }

        singularity_message(DEBUG, "Send JSON runtime configuration to scontainer stage 1\n");

        /* send json configuration to scontainer */
        if ( write(input[1], json_stdin, config.jsonConfSize) != config.jsonConfSize ) {
            singularity_message(ERROR, "Copy json configuration failed\n");
            exit(1);
        }

        singularity_message(DEBUG, "Wait C and JSON runtime configuration from scontainer stage 1\n");

        while ( poll(&fds, 1, -1) >= 0 ) {
            if ( fds.revents == POLLIN ) {
                int ret;
                singularity_message(DEBUG, "Receiving configuration from scontainer stage 1\n");
                if ( (ret = read(output[0], &config, sizeof(config))) != sizeof(config) ) {
                    singularity_message(ERROR, "Failed to read communication pipe %d\n", ret);
                    exit(1);
                }
                if ( config.jsonConfSize >= MAX_JSON_SIZE) {
                    singularity_message(ERROR, "json configuration too big\n");
                    exit(1);
                }
                if ( (ret = read(output[0], json_stdin, config.jsonConfSize)) != config.jsonConfSize ) {
                    singularity_message(ERROR, "Failed to read communication pipe %d\n", ret);
                    exit(1);
                }
                json_stdin[config.jsonConfSize] = '\0';
            }
            break;
        }

        close(output[0]);
        close(input[1]);

        singularity_message(DEBUG, "Wait completion of scontainer stage1\n");
        if ( wait(&status) != stage1 ) {
            singularity_message(ERROR, "Can't wait child\n");
            exit(1);
        }

        if ( WIFEXITED(status) || WIFSIGNALED(status) ) {
            if ( WEXITSTATUS(status) != 0 ) {
                singularity_message(ERROR, "Child exit with status %d\n", WEXITSTATUS(status));
                exit(1);
            }
        }
        close(stage_socket[0]);

        /* block SIGCHLD signal handled later by scontainer/smaster */
        singularity_message(DEBUG, "Set child signal mask\n");
        sigemptyset(&mask);
        sigaddset(&mask, SIGCHLD);
        if (sigprocmask(SIG_SETMASK, &mask, NULL) == -1) {
            singularity_message(ERROR, "Blocked signals error\n");
            exit(1);
        }

        if ( config.isInstance ) {
            singularity_message(VERBOSE, "Run as instance\n");
            int forked = fork();
            if ( forked == 0 ) {
                int i;
                if ( chdir("/") < 0 ) {
                    singularity_message(ERROR, "Can't change directory to /\n");
                    exit(1);
                }
                if ( setsid() < 0 ) {
                    singularity_message(ERROR, "Can't set session leader\n");
                    exit(1);
                }
                umask(0);

                singularity_message(DEBUG, "Close all file descriptor\n");
                for( i = sysconf(_SC_OPEN_MAX); i > 2; i-- ) {
                    close(i);
                }
            } else {
                int status;

                singularity_message(DEBUG, "Wait child process signaling SIGSTOP\n");
                waitpid(forked, &status, WUNTRACED);
                if ( WIFSTOPPED(status) ) {
                    singularity_message(DEBUG, "Send SIGCONT to child process\n");
                    kill(forked, SIGCONT);
                    return(0);
                }
                if ( WIFEXITED(status) || WIFSIGNALED(status) ) {
                    singularity_message(VERBOSE, "Child process exited with status %d\n", WEXITSTATUS(status));
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

        singularity_message(DEBUG, "Create socketpair communication between smaster and scontainer\n");
        if ( socketpair(AF_UNIX, SOCK_STREAM, 0, stage_socket) < 0 ) {
            singularity_message(ERROR, "Failed to create communication socket\n");
            exit(1);
        }

        if ( pipe2(input, 0) < 0 ) {
            singularity_message(ERROR, "failed to create input pipes\n");
            exit(1);
        }

        /* enforce PID namespace if NO_NEW_PRIVS not supported  */
        if ( config.hasNoNewPrivs == 0 ) {
            singularity_message(VERBOSE, "No PR_SET_NO_NEW_PRIVS support, enforcing PID namespace\n");
            config.nsFlags |= CLONE_NEWPID;
        }

        if ( config.pidPid ) {
            enter_namespace(config.pidPid, CLONE_NEWPID);
            stage2 = fork();
        } else {
            if ( config.nsFlags & CLONE_NEWPID ) {
                singularity_message(VERBOSE, "Create pid namespace\n");
                stage2 = fork_ns(CLONE_NEWPID);
            } else {
                stage2 = fork();
            }
        }

        if ( stage2 == 0 ) {
            /* at this stage we are PID 1 if PID namespace requested */
            int rpc_socket[2];
            pid_t child;

            singularity_message(VERBOSE, "Spawn scontainer stage 2\n");

            set_parent_death_signal(SIGKILL);

            if ( config.netPid ) {
                enter_namespace(config.netPid, CLONE_NEWNET);
            } else {
                if ( config.nsFlags & CLONE_NEWNET ) {
                    singularity_message(VERBOSE, "Create net namespace\n");
                    if ( unshare(CLONE_NEWNET) < 0 ) {
                        singularity_message(ERROR, "failed to create network namespace\n");
                        exit(1);
                    }
                }
            }
            if ( config.utsPid ) {
                enter_namespace(config.utsPid, CLONE_NEWUTS);
            } else {
                if ( config.nsFlags & CLONE_NEWUTS ) {
                    singularity_message(VERBOSE, "Create uts namespace\n");
                    if ( unshare(CLONE_NEWUTS) < 0 ) {
                        singularity_message(ERROR, "failed to create uts namespace\n");
                        exit(1);
                    }
                }
            }
            if ( config.ipcPid ) {
                enter_namespace(config.ipcPid, CLONE_NEWIPC);
            } else {
                if ( config.nsFlags & CLONE_NEWIPC ) {
                    singularity_message(VERBOSE, "Create ipc namespace\n");
                    if ( unshare(CLONE_NEWIPC) < 0 ) {
                        singularity_message(ERROR, "failed to create ipc namespace\n");
                        exit(1);
                    }
                }
            }
#ifdef CLONE_NEWCGROUP
            if ( config.cgroupPid ) {
                enter_namespace(config.cgroupPid, CLONE_NEWCGROUP);
            } else {
                if ( config.nsFlags & CLONE_NEWCGROUP ) {
                    singularity_message(VERBOSE, "Create cgroup namespace\n");
                    if ( unshare(CLONE_NEWCGROUP) < 0 ) {
                        singularity_message(ERROR, "failed to create cgroup namespace\n");
                        exit(1);
                    }
                }
            }
#endif
            if ( config.mntPid ) {
                enter_namespace(config.mntPid, CLONE_NEWNS);
            } else {
                singularity_message(VERBOSE, "Unshare filesystem and create mount namespace\n");
                if ( unshare(CLONE_FS) < 0 ) {
                    singularity_message(ERROR, "Failed to unshare filesystem\n");
                    exit(1);
                }
                if ( unshare(CLONE_NEWNS) < 0 ) {
                    singularity_message(ERROR, "Failed to unshare mount namespace\n");
                    exit(1);
                }
            }

            singularity_message(DEBUG, "Create RPC socketpair for communication between scontainer and RPC server\n");
            if ( socketpair(AF_UNIX, SOCK_STREAM, 0, rpc_socket) < 0 ) {
                singularity_message(ERROR, "Failed to create communication socket\n");
                exit(1);
            }

            close(stage_socket[0]);
            close(input[1]);

            child = fork();
            if ( child == 0 ) {
                void *handle;
                GoInt (*rpcserver)(GoInt socket);

                close(input[0]);

                singularity_message(VERBOSE, "Spawn RPC server\n");

                close(stage_socket[1]);
                close(rpc_socket[0]);

                /* return to host network namespace for network setup */
                singularity_message(DEBUG, "Return to host network namespace\n");
                if ( config.nsFlags & CLONE_NEWNET && (config.nsFlags & CLONE_NEWUSER) == 0 ) {
                    enter_namespace(parent, CLONE_NEWNET);
                }

                /* Use setfsuid to address issue about root_squash filesystems option */
                if ( config.isSuid ) {
                    if ( setfsuid(uid) < 0 ) {
                        singularity_message(ERROR, "Failed to set fs uid\n");
                        exit(1);
                    }
                }

                /*
                 * If we execute rpc server there, we will lose all capabilities during execve
                 * when using user namespace, so we won't be able to serve any privileged operations,
                 * a solution is to load rpc server as a shared library
                 */
                singularity_message(DEBUG, "Load " BUILDDIR "/librpc.so\n");
                handle = dlopen(BUILDDIR "/librpc.so", RTLD_LAZY);
                if ( handle == NULL ) {
                    singularity_message(ERROR, "Failed to load shared lib librpc.so\n");
                    exit(1);
                }
                rpcserver = (GoInt (*)(GoInt))dlsym(handle, "RPCServer");
                if ( rpcserver == NULL ) {
                    singularity_message(ERROR, "Failed to find symbol\n");
                    exit(1);
                }

                free(json_stdin);

                singularity_message(VERBOSE, "Serve RPC requests\n");

                return(rpcserver((GoInt)rpc_socket[1]));
            } else if ( child > 0 ) {
                setenv("SCONTAINER_STAGE", "2", 1);
                setenv("SCONTAINER_SOCKET", int2str(stage_socket[1]), 1);
                setenv("SCONTAINER_RPC_SOCKET", int2str(rpc_socket[0]), 1);

                if ( dup2(input[0], JOKER) < 0 ) {
                    singularity_message(ERROR, "failed to create stdin pipe\n");
                    exit(1);
                }
                close(input[0]);
                close(rpc_socket[1]);

                singularity_message(VERBOSE, "Execute scontainer stage 2\n");
                execle(BUILDDIR "/scontainer", BUILDDIR "/scontainer", NULL, environ);
            }
            singularity_message(ERROR, "Failed to execute container\n");
            exit(1);
        } else if ( stage2 > 0 ) {
            setenv("SMASTER_CONTAINER_PID", int2str(stage2), 1);
            setenv("SMASTER_SOCKET", int2str(stage_socket[0]), 1);

            close(input[0]);

            config.containerPid = stage2;

            singularity_message(VERBOSE, "Spawn smaster process\n");

            close(stage_socket[1]);

            /* send runtime configuration to scontainer (CGO) */
            singularity_message(DEBUG, "Send C runtime configuration to scontainer stage 2\n");
            if ( write(input[1], &config, sizeof(config)) != sizeof(config) ) {
                singularity_message(ERROR, "failed to send runtime configuration\n");
                exit(1);
            }

            /* send json configuration to scontainer */
            singularity_message(DEBUG, "Send JSON runtime configuration to scontainer stage 2\n");
            if ( write(input[1], json_stdin, config.jsonConfSize) != config.jsonConfSize ) {
                singularity_message(ERROR, "copy json configuration failed\n");
                exit(1);
            }

            singularity_message(VERBOSE, "Execute smaster process\n");
            execle(BUILDDIR "/smaster", BUILDDIR "/smaster", NULL, environ);
        }
        singularity_message(ERROR, "Failed to create container namespaces\n");
        exit(1);
    }
    return(0);
}
