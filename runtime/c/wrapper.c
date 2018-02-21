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
#include <sys/eventfd.h>
#include <setjmp.h>
#include <sys/signalfd.h>
#include <linux/securebits.h>
#include <linux/capability.h>
#include <sys/syscall.h>
#include <dlfcn.h>

#include "wrapper.h"
#include "librpc.h"

#define CLONE_STACK_SIZE    1024*1024
#define MAX_JSON_SIZE       64*1024
#define BUFSIZE             512

int rpc = 0;

extern char **environ;

typedef struct fork_state_s {
    sigjmp_buf env;
} fork_state_t;

int setns(int fd, int nstype) {
    return syscall(__NR_setns, fd, nstype);
}

static void pfatal(const char *fmt, ...) {
    char buffer[BUFSIZE];
    va_list arg;

    memset(buffer, 0, BUFSIZE);

    va_start(arg, fmt);
    vsnprintf(buffer, BUFSIZE-1, fmt, arg);
    va_end(arg);

    fprintf(stderr, "error: %s\n", buffer);

    exit(1);
}

static const char *int2str(int num) {
    char *str = (char *)malloc(16);
    memset(str, 0, 16);
    snprintf(str, 15, "%d", num);
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
    if ( seteuid(0) < 0 || setegid(0) < 0 ) {
        pfatal("Failed to set effective UID/GID to 0");
    }
}

static void enter_namespace(pid_t pid, int nstype) {
    int ns_fd;
    char buffer[256];
    char *namespace = NULL;

    if ( nstype & CLONE_NEWPID ) {
        namespace = strdup("pid");
    } else if ( nstype & CLONE_NEWNET ) {
        namespace = strdup("net");
    } else if ( nstype & CLONE_NEWIPC ) {
        namespace = strdup("ipc");
    } else if ( nstype & CLONE_NEWNS ) {
        namespace = strdup("mnt");
    } else if ( nstype & CLONE_NEWUTS ) {
        namespace = strdup("uts");
    } else if ( nstype & CLONE_NEWUSER ) {
        namespace = strdup("user");
    }

    memset(buffer, 0, 256);
    snprintf(buffer, 255, "/proc/%d/ns/%s", pid, namespace);

    ns_fd = open(buffer, O_RDONLY);
    if ( ns_fd < 0 ) {
        pfatal("Failed to enter in namespace %s of PID %d: %s", namespace, pid, strerror(errno));
    }

    if ( setns(ns_fd, nstype) < 0 ) {
        pfatal("Failed to enter in namespace %s of PID %d: %s", namespace, pid, strerror(errno));
    }

    close(ns_fd);
    free(namespace);
}

static void setup_userns(const struct uidMapping uidMapping, const struct gidMapping gidMapping) {
    FILE *map_fp;
    uid_t containerUid = uidMapping.containerID;
    uid_t hostUid = uidMapping.hostID;
    gid_t containerGid = gidMapping.containerID;
    gid_t hostGid = gidMapping.hostID;

    if ( unshare(CLONE_NEWUSER) < 0 ) {
        pfatal("Failed to create user namespace");
    }

    map_fp = fopen("/proc/self/setgroups", "w+"); // Flawfinder: ignore
    if ( map_fp != NULL ) {
        fprintf(map_fp, "deny\n");
        if ( fclose(map_fp) < 0 ) {
            pfatal("Failed to write deny to setgroup file: %s\n", strerror(errno));
        }
    } else {
        pfatal("Could not write info to setgroups: %s\n", strerror(errno));
    }

    map_fp = fopen("/proc/self/gid_map", "w+"); // Flawfinder: ignore
    if ( map_fp != NULL ) {
        fprintf(map_fp, "%i %i %i\n", containerGid, hostGid, gidMapping.size);
        if ( fclose(map_fp) < 0 ) {
            pfatal("Failed to write to GID map: %s\n", strerror(errno));
        }
    } else {
        pfatal("Could not write parent info to gid_map: %s\n", strerror(errno));
    }

    map_fp = fopen("/proc/self/uid_map", "w+"); // Flawfinder: ignore
    if ( map_fp != NULL ) {
        fprintf(map_fp, "%i %i %i\n", containerUid, hostUid, uidMapping.size);
        if ( fclose(map_fp) < 0 ) {
            pfatal("Failed to write to UID map: %s\n", strerror(errno));
        }
    } else {
        pfatal("Could not write parent info to uid_map: %s\n", strerror(errno));
    }
}

static unsigned char is_suid(void) {
    ElfW(auxv_t) *auxv;
    unsigned char suid = 0;
    char *progname = NULL;
    char *buffer = (char *)malloc(4096);
    int proc_auxv = open("/proc/self/auxv", O_RDONLY);

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
    if ( prctl(PR_SET_PDEATHSIG, signo) < 0 ) {
        pfatal("Failed to set parent death signal");
    }
}

void do_nothing(int sig) {
    return;
}

//int main(int argc, char **argv) {
__attribute__((constructor)) static int init(int argc, char **argv) {
    char *json_stdin;
    int stage_socket[2];
    pid_t stage1, stage2;
    uid_t uid = getuid();
    gid_t gid = getgid();
    struct cConfig config;
    sigset_t mask;

    memset(&config, 0, sizeof(config));

    config.isSuid = is_suid();

    if ( config.isSuid ) {
        if ( setegid(gid) < 0 || seteuid(uid) < 0 ) {
            pfatal("Failed to drop privileges");
        }
    }

    /* don't deal with environment variables in C and Go */
    environ = NULL;

#ifdef PR_SET_NO_NEW_PRIVS
    config.hasNoNewPrivs = 1;
#else
    config.hasNoNewPrivs = 0;
#endif

    /* read json configuration from stdin */
    int std = open("/proc/self/fd/1", O_RDONLY);

    json_stdin = (char *)malloc(MAX_JSON_SIZE);
    if ( json_stdin == NULL ) {
        pfatal("memory allocation failure");
    }

    memset(json_stdin, 0, MAX_JSON_SIZE);
    if ( ( config.jsonConfSize = read(STDIN_FILENO, json_stdin, MAX_JSON_SIZE) ) < 0 ) {
        pfatal("Read from stdin failed");
    }

    /* back to terminal stdin */
    if ( isatty(std) ) {
        dup2(std, 0);
    }
    close(std);

    signal(SIGCHLD, &do_nothing);

    /* for security reasons use socketpair only for process communications */
    if ( socketpair(AF_UNIX, SOCK_DGRAM, 0, stage_socket) < 0 ) {
        pfatal("Failed to create communication socket");
    }

    stage1 = fork();
    if ( stage1 == 0 ) {
        char **env;

        environ = env;
        setenv("STAGE", "1", 1);
        setenv("SOCKET", int2str(stage_socket[1]), 1);

        close(stage_socket[0]);

        /*
         *  stage1 is responsible for singularity configuration file parsing, handle user input,
         *  read capabilities, check what namespaces is required.
         */
        if ( config.isSuid ) {
            priv_escalate();
        }

        execle("/tmp/scontainer", "/tmp/scontainer", "-stage", "1", "-socket", int2str(stage_socket[1]), NULL, environ);
        pfatal("Failed to execute scontainer");
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

        /* send runtime configuration to scontainer (CGO) */
        if ( write(stage_socket[0], &config, sizeof(config)) != sizeof(config) ) {
            pfatal("copy failed %d", sizeof(config));
        }

        /* send json configuration to scontainer */
        if ( write(stage_socket[0], json_stdin, config.jsonConfSize) != config.jsonConfSize ) {
            pfatal("copy json configuration failed");
        }

        while ( poll(&fds, 1, -1) >= 0 ) {
            if ( fds.revents == POLLIN ) {
                int ret;
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
        sigemptyset(&mask);
        sigaddset(&mask, SIGCHLD);
        if (sigprocmask(SIG_SETMASK, &mask, NULL) == -1) {
            pfatal("blocked signals error");
        }

        if ( config.isInstance ) {
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
                for( i = sysconf(_SC_OPEN_MAX); i > 2; i-- ) {
                    close(i);
                }
            } else {
                int status;
                waitpid(forked, &status, WUNTRACED);
                if ( WIFSTOPPED(status) ) {
                    kill(forked, SIGCONT);
                    return(0);
                }
                if ( WIFEXITED(status) || WIFSIGNALED(status) ) {
                    return(WEXITSTATUS(status));
                }
                return(-1);
            }
        }

        if ( config.userNS == 0 ) {
            priv_escalate();
        } else {
            if ( config.userPid ) {
                enter_namespace(config.userPid, CLONE_NEWUSER);
            } else {
                setup_userns(config.uidMapping, config.gidMapping);
            }
        }

        if ( socketpair(AF_UNIX, SOCK_STREAM, 0, stage_socket) < 0 ) {
            pfatal("Failed to create communication socket");
        }

        /* enforce PID namespace if NO_NEW_PRIVS not supported  */
        if ( config.hasNoNewPrivs == 0 ) {
            config.nsFlags |= CLONE_NEWPID;
        }

        if ( config.pidPid ) {
            enter_namespace(config.pidPid, CLONE_NEWPID);
            stage2 = fork();
        } else {
            if ( config.nsFlags & CLONE_NEWPID ) {
                stage2 = fork_ns(CLONE_NEWPID);
            } else {
                stage2 = fork();
            }
        }

        if ( stage2 == 0 ) {
            /* at this stage we are PID 1 if PID namespace requested */
            int rpc_socket[2];
            pid_t child;

            set_parent_death_signal(SIGKILL);

            if ( config.netPid ) {
                enter_namespace(config.netPid, CLONE_NEWNET);
            } else {
                if ( config.nsFlags & CLONE_NEWNET && unshare(CLONE_NEWNET) < 0 ) {
                    pfatal("failed to create network namespace");
                }
            }
            if ( config.utsPid ) {
                enter_namespace(config.utsPid, CLONE_NEWUTS);
            } else {
                if ( config.nsFlags & CLONE_NEWUTS && unshare(CLONE_NEWUTS) < 0 ) {
                    pfatal("failed to create uts namespace");
                }
            }
            if ( config.ipcPid ) {
                enter_namespace(config.ipcPid, CLONE_NEWIPC);
            } else { 
                if ( config.nsFlags & CLONE_NEWIPC && unshare(CLONE_NEWIPC) < 0 ) {
                    pfatal("failed to create ipc namespace");
                }
            }
#ifdef CLONE_NEWCGROUP
            if ( config.cgroupPid ) {
                enter_namespace(config.cgroupPid, CLONE_NEWCGROUP);
            } else { 
                if ( config.nsFlags & CLONE_NEWCGROUP && unshare(CLONE_NEWCGROUP) < 0 ) {
                    pfatal("failed to create cgroup namespace");
                }
            }
#endif
            if ( config.mntPid ) {
                enter_namespace(config.mntPid, CLONE_NEWNS);
            } else {
                if ( unshare(CLONE_FS) < 0 ) {
                    pfatal("Failed to unshare filesystem");
                }
                if ( unshare(CLONE_NEWNS) < 0 ) {
                    pfatal("Failed to unshare mount namespace");
                }
            }

            if ( socketpair(AF_UNIX, SOCK_STREAM, 0, rpc_socket) < 0 ) {
                pfatal("Failed to create communication socket");
            }

            close(stage_socket[0]);

            child = fork();
            if ( child == 0 ) {
/*                void *handle;
                void (*rpcserver)(GoInt socket);
*/
                close(stage_socket[1]);
                close(rpc_socket[0]);

                /* return to host network namespace for network setup */
                if ( config.nsFlags & CLONE_NEWNET && config.userNS == 0 ) {
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
/*                handle = dlopen("/tmp/librpc.so", RTLD_LAZY);
                if ( handle == NULL ) {
                    pfatal("Failed to load shared lib librpc.so");
                }
                rpcserver = (void (*)(GoInt))dlsym(handle, "RpcServer");
                if ( rpcserver == NULL ) {
                    pfatal("Failed to find symbol");
                }
*/
                free(json_stdin);

  //              rpcserver((GoInt)rpc_socket[1]);
                //RpcServer((GoInt)rpc_socket[1]);
                rpc = rpc_socket[1];

                return(0);
            } else if ( child > 0 ) {
                char **env;

                environ = env;
                setenv("STAGE", "2", 1);
                setenv("SOCKET", int2str(stage_socket[1]), 1);

                close(rpc_socket[1]);

                /* send json configuration to smaster */
                if ( write(stage_socket[1], json_stdin, config.jsonConfSize) != config.jsonConfSize ) {
                    pfatal("copy json configuration failed");
                }

                execle("/tmp/scontainer", "/tmp/scontainer", "-stage", "2", "-socket", int2str(stage_socket[1]), "-rpc", int2str(rpc_socket[0]), NULL, environ);
            }
            pfatal("Failed to execute container");
        } else if ( stage2 > 0 ) {
            config.containerPid = stage2;

            close(stage_socket[1]);

            /* send runtime configuration to scontainer (CGO) */
            if ( write(stage_socket[0], &config, sizeof(config)) != sizeof(config) ) {
                pfatal("copy failed %d", sizeof(config));
            }

            /* send json configuration to scontainer */
            if ( write(stage_socket[0], json_stdin, config.jsonConfSize) != config.jsonConfSize ) {
                pfatal("copy json configuration failed");
            }

            execl("/tmp/smaster", "/tmp/smaster", int2str(stage2), int2str(stage_socket[0]), NULL);
        }
        pfatal("Failed to create container namespaces");
    }
    return(0);
}

int main() {
    if (rpc) {
        RpcServer((GoInt)rpc);
    }
}
