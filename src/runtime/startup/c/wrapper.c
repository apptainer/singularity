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
#include <limits.h>
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
#include <sys/eventfd.h>

#ifdef SINGULARITY_SECUREBITS
#  include <linux/securebits.h>
#else
#  include "c/lib/util/securebits.h"
#endif /* SINGULARITY_SECUREBITS */

#ifndef PR_SET_NO_NEW_PRIVS
#define PR_SET_NO_NEW_PRIVS 38
#endif

#ifndef PR_GET_NO_NEW_PRIVS
#define PR_GET_NO_NEW_PRIVS 39
#endif

#ifndef CLONE_NEWUSER
#define CLONE_NEWUSER       0x10000000
#endif

#ifndef CLONE_NEWCGROUP
#define CLONE_NEWCGROUP     0x02000000
#endif

#include "c/lib/util/capability.h"
#include "c/lib/util/message.h"

#include "startup/c/wrapper.h"

#define CLONE_STACK_SIZE    1024*1024
#define BUFSIZE             512

extern char **environ;

/* C and JSON configuration */
struct cConfig config;
char *json_stdin;

/* Socket process communication */
int rpc_socket[2] = {-1, -1};
int master_socket[2] = {-1, -1};

#define SCONTAINER_STAGE1   1
#define SCONTAINER_STAGE2   2
#define SMASTER             4
#define RPC_SERVER          5

unsigned char execute = SCONTAINER_STAGE1;
pid_t stage_pid;
char *sruntime;

typedef struct fork_state_s {
    sigjmp_buf env;
} fork_state_t;

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

    int retval = clone(clone_fn, child_stack_ptr, (SIGCHLD|flags), &state);
    return retval;
}

static void priv_escalate(void) {
    singularity_message(VERBOSE, "Get root privileges\n");
    if ( seteuid(0) < 0 ) {
        singularity_message(ERROR, "Failed to set effective UID to 0\n");
        exit(1);
    }
}

static void set_parent_death_signal(int signo) {
    singularity_message(DEBUG, "Set parent death signal to %d\n", signo);
    if ( prctl(PR_SET_PDEATHSIG, signo) < 0 ) {
        singularity_message(ERROR, "Failed to set parent death signal\n");
        exit(1);
    }
}

static void prepare_scontainer_stage(int stage) {
    uid_t uid = getuid();
    struct __user_cap_header_struct header;
    struct __user_cap_data_struct data[2];

    set_parent_death_signal(SIGKILL);

    singularity_message(DEBUG, "Entering in scontainer stage %d\n", stage);

    execute = stage;

    header.version = LINUX_CAPABILITY_VERSION;
    header.pid = 0;

    if ( capget(&header, data) < 0 ) {
        singularity_message(ERROR, "Failed to get processus capabilities\n");
        exit(1);
    }

    data[1].inheritable = (__u32)(config.capInheritable >> 32);
    data[0].inheritable = (__u32)(config.capInheritable & 0xFFFFFFFF);
    data[1].permitted = (__u32)(config.capPermitted >> 32);
    data[0].permitted = (__u32)(config.capPermitted & 0xFFFFFFFF);
    data[1].effective = (__u32)(config.capEffective >> 32);
    data[0].effective = (__u32)(config.capEffective & 0xFFFFFFFF);

    int last_cap;
    for ( last_cap = CAPSET_MAX; ; last_cap-- ) {
        if ( prctl(PR_CAPBSET_READ, last_cap) > 0 || last_cap == 0 ) {
            break;
        }
    }

    int caps_index;
    for ( caps_index = 0; caps_index <= last_cap; caps_index++ ) {
        if ( !(config.capBounding & (1ULL << caps_index)) ) {
            if ( prctl(PR_CAPBSET_DROP, caps_index) < 0 ) {
                singularity_message(ERROR, "Failed to drop bounding capabilities set: %s\n", strerror(errno));
                exit(1);
            }
        }
    }

    if ( config.isSuid && !(config.nsFlags & CLONE_NEWUSER) ) {
        if ( prctl(PR_SET_SECUREBITS, SECBIT_NO_SETUID_FIXUP|SECBIT_NO_SETUID_FIXUP_LOCKED) < 0 ) {
            singularity_message(ERROR, "Failed to set securebits: %s\n", strerror(errno));
            exit(1);
        }

        if ( setresuid(uid, uid, uid) < 0 ) {
            singularity_message(ERROR, "Faile to drop privileges: %s\n", strerror(errno));
            exit(1);
        }

        set_parent_death_signal(SIGKILL);
    }

    if ( config.noNewPrivs ) {
        if ( prctl(PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0) < 0 ) {
            singularity_message(ERROR, "Failed to set no new privs flag: %s\n", strerror(errno));
            exit(1);
        }
        if ( prctl(PR_GET_NO_NEW_PRIVS, 0, 0 ,0, 0) != 1 ) {
            singularity_message(ERROR, "Aborting, failed to set no new privs flag: %s\n", strerror(errno));
            exit(1);
        }
    }

    if ( capset(&header, data) < 0 ) {
        singularity_message(ERROR, "Failed to set process capabilities\n");
        exit(1);
    }

#ifdef USER_CAPABILITIES
    // set ambient capabilities if supported
    for ( caps_index = 0; caps_index <= last_cap; caps_index++ ) {
        if ( (config.capAmbient & (1ULL << caps_index)) ) {
            if ( prctl(PR_CAP_AMBIENT, PR_CAP_AMBIENT_RAISE, caps_index, 0, 0) < 0 ) {
                singularity_message(ERROR, "Failed to set ambient capability: %s\n", strerror(errno));
                exit(1);
            }
        }
    }
#endif
}

static int create_namespace(int nstype) {
    switch(nstype) {
    case CLONE_NEWNET:
#ifdef NS_CLONE_NEWNET
        singularity_message(VERBOSE, "Create network namespace\n");
#else
        singularity_message(WARNING, "Skipping network namespace creation, not supported\n");
        return(0);
#endif /* NS_CLONE_NEWNET */
        break;
    case CLONE_NEWIPC:
#ifdef NS_CLONE_NEWIPC
        singularity_message(VERBOSE, "Create ipc namespace\n");
#else
        singularity_message(WARNING, "Skipping ipc namespace creation, not supported\n");
        return(0);
#endif /* NS_CLONE_NEWIPC */
        break;
    case CLONE_NEWNS:
#ifdef NS_CLONE_NEWNS
        singularity_message(VERBOSE, "Create mount namespace\n");
#else
        singularity_message(WARNING, "Skipping mount namespace creation, not supported\n");
        return(0);
#endif /* NS_CLONE_NEWNS */
        break;
    case CLONE_NEWUTS:
#ifdef NS_CLONE_NEWUTS
        singularity_message(VERBOSE, "Create uts namespace\n");
#else
        singularity_message(WARNING, "Skipping uts namespace creation, not supported\n");
        return(0);
#endif /* NS_CLONE_NEWUTS */
        break;
    case CLONE_NEWUSER:
#ifdef NS_CLONE_NEWUSER
        singularity_message(VERBOSE, "Create user namespace\n");
#else
        singularity_message(WARNING, "Skipping user namespace creation, not supported\n");
#endif /* NS_CLONE_NEWUSER */
        break;
#ifdef NS_CLONE_NEWCGROUP
    case CLONE_NEWCGROUP:
        singularity_message(VERBOSE, "Create cgroup namespace\n");
        break;
#endif /* NS_CLONE_NEWCGROUP */
    default:
        singularity_message(WARNING, "Skipping unknown namespace creation\n");
        errno = EINVAL;
        return(-1);
    }
    return unshare(nstype);
}

static int enter_namespace(pid_t pid, int nstype) {
    int ns_fd;
    static char buffer[PATH_MAX];
    char *namespace;

    memset(buffer, 0, PATH_MAX);

    switch(nstype) {
    case CLONE_NEWPID:
        singularity_message(VERBOSE, "Entering in pid namespace\n");
#ifdef NS_CLONE_NEWPID
        namespace = strdup("pid");
#else
        errno = EINVAL;
        return(-1);
#endif /* NS_CLONE_NEWPID */
        break;
    case CLONE_NEWNET:
        singularity_message(VERBOSE, "Entering in network namespace\n");
#ifdef NS_CLONE_NEWNET
        namespace = strdup("net");
#else
        errno = EINVAL;
        return(-1);
#endif /* NS_CLONE_NEWNET */
        break;
    case CLONE_NEWIPC:
        singularity_message(VERBOSE, "Entering in ipc namespace\n");
#ifdef NS_CLONE_NEWIPC
        namespace = strdup("ipc");
#else
        errno = EINVAL;
        return(-1);
#endif /* NS_CLONE_NEWIPC */
        break;
    case CLONE_NEWNS:
        singularity_message(VERBOSE, "Entering in mount namespace\n");
#ifdef NS_CLONE_NEWNS
        namespace = strdup("mnt");
#else
        errno = EINVAL;
        return(-1);
#endif /* NS_CLONE_NEWNS */
        break;
    case CLONE_NEWUTS:
        singularity_message(VERBOSE, "Entering in uts namespace\n");
#ifdef NS_CLONE_NEWUTS
        namespace = strdup("uts");
#else
        errno = EINVAL;
        return(-1);
#endif /* NS_CLONE_NEWUTS */
        break;
    case CLONE_NEWUSER:
        singularity_message(VERBOSE, "Entering in user namespace\n");
#ifdef NS_CLONE_NEWUSER
        namespace = strdup("user");
#else
        errno = EINVAL;
        return(-1);
#endif /* NS_CLONE_NEWUSER */
        break;
#ifdef NS_CLONE_NEWCGROUP
    case CLONE_NEWCGROUP:
        singularity_message(VERBOSE, "Entering in cgroup namespace\n");
        namespace = strdup("cgroup");
        break;
#endif /* NS_CLONE_NEWCGROUP */
    default:
        singularity_message(VERBOSE, "Entering in unknown namespace\n");
        errno = EINVAL;
        return(-1);
    }

    snprintf(buffer, PATH_MAX-1, "/proc/%d/ns/%s", pid, namespace);
    singularity_message(DEBUG, "Opening namespace file descriptor %s\n", buffer);
    ns_fd = open(buffer, O_RDONLY);
    if ( ns_fd < 0 ) {
        return(-1);
    }

    if ( setns(ns_fd, nstype) < 0 ) {
        int err = errno;
        close(ns_fd);
        free(namespace);
        errno = err;
        return(-1);
    }

    close(ns_fd);
    free(namespace);
    return(0);
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
    if ( read(proc_auxv, buffer, 4088) < 0 ) {
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

void do_exit(int sig) {
    if ( sig == SIGUSR1 ) {
        exit(0);
    }
    exit(1);
}

__attribute__((constructor)) static void init(void) {
    char *env[8] = {0};
    uid_t uid = getuid();
    gid_t gid = getgid();
    sigset_t mask;
    char *loglevel;
    char *runtime;
    char *pipe_fd_env;
    int output[2];
    int status;
    struct pollfd fds[2];
    int syncfd = -1;
    int pipe_fd = -1;

#ifndef SINGULARITY_NO_NEW_PRIVS
    singularity_message(ERROR, "Host kernel is outdated and does not support PR_SET_NO_NEW_PRIVS!\n");
    exit(1);
#endif

    loglevel = getenv("SINGULARITY_MESSAGELEVEL");
    if ( loglevel != NULL ) {
        loglevel = strdup(loglevel);
    } else {
        singularity_message(ERROR, "SINGULARITY_MESSAGELEVEL environment variable isn't set\n");
        exit(1);
    }

    runtime = getenv("SRUNTIME");
    if ( runtime != NULL ) {
        sruntime = strdup(runtime);
    } else {
        singularity_message(ERROR, "SRUNTIME environment variable isn't set\n");
        exit(1);
    }

    pipe_fd_env = getenv("PIPE_EXEC_FD");
    if ( pipe_fd_env != NULL ) {
        if ( sscanf(pipe_fd_env, "%d", &pipe_fd) != 1 ) {
            singularity_message(ERROR, "Failed to parse PIPE_EXEC_FD environment variable: %s\n", strerror(errno));
            exit(1);
        }
        singularity_message(DEBUG, "PIPE_EXEC_FD value: %d\n", pipe_fd);
        if ( pipe_fd < 0 || pipe_fd >= sysconf(_SC_OPEN_MAX) ) {
            singularity_message(ERROR, "Bad PIPE_EXEC_FD file descriptor value\n");
            exit(1);
        }
    } else {
        singularity_message(ERROR, "PIPE_EXEC_FD environment variable isn't set\n");
        exit(1);
    }

    singularity_message(VERBOSE, "Container runtime\n");

    memset(&config, 0, sizeof(config));

    config.isSuid = is_suid();

    if ( config.isSuid || geteuid() == 0 ) {
        /* force kernel to load overlay module to ease detection later */
        if ( mount("none", "/", "overlay", MS_SILENT, "") < 0 ) {
            if ( errno != EINVAL ) {
                singularity_message(DEBUG, "Overlay seems not supported by kernel\n");
            } else {
                singularity_message(DEBUG, "Overlay seems supported by kernel\n");
            }
        }
    }

    if ( config.isSuid ) {
        singularity_message(DEBUG, "Drop privileges\n");
        if ( setegid(gid) < 0 || seteuid(uid) < 0 ) {
            singularity_message(ERROR, "Failed to drop privileges: %s\n", strerror(errno));
            exit(1);
        }
    }

    /* reset environment variables */
    clearenv();

    if ( loglevel != NULL ) {
        setenv("SINGULARITY_MESSAGELEVEL", loglevel, 1);
        free(loglevel);
    }

    /* read json configuration from stdin */
    singularity_message(DEBUG, "Read json configuration from pipe\n");

    json_stdin = (char *)malloc(MAX_JSON_SIZE);
    if ( json_stdin == NULL ) {
        singularity_message(ERROR, "Memory allocation failed: %s\n", strerror(errno));
        exit(1);
    }

    memset(json_stdin, 0, MAX_JSON_SIZE);
    if ( ( config.jsonConfSize = read(pipe_fd, json_stdin, MAX_JSON_SIZE - 1) ) <= 0 ) {
        singularity_message(ERROR, "Read JSON configuration from pipe failed: %s\n", strerror(errno));
        exit(1);
    }
    close(pipe_fd);

    /* block SIGCHLD signal handled later by scontainer/smaster */
    singularity_message(DEBUG, "Set child signal mask\n");
    sigemptyset(&mask);
    sigaddset(&mask, SIGCHLD);
    if (sigprocmask(SIG_SETMASK, &mask, NULL) == -1) {
        singularity_message(ERROR, "Blocked signals error: %s\n", strerror(errno));
        exit(1);
    }

    if ( pipe2(output, 0) < 0 ) {
        singularity_message(ERROR, "Failed to create output process pipes: %s\n", strerror(errno));
        exit(1);
    }

    singularity_message(DEBUG, "Create socketpair for smaster communication channel\n");
    if ( socketpair(AF_UNIX, SOCK_STREAM|SOCK_CLOEXEC, 0, master_socket) < 0 ) {
        singularity_message(ERROR, "Failed to create communication socket: %s\n", strerror(errno));
        exit(1);
    }

    stage_pid = fork();
    if ( stage_pid == 0 ) {
        set_parent_death_signal(SIGKILL);

        close(output[0]);
        close(master_socket[0]);

        if ( dup2(output[1], STDOUT_FILENO) < 0 ) {
            singularity_message(ERROR, "Failed to create stdout pipe: %s\n", strerror(errno));
            exit(1);
        }
        close(output[1]);

        singularity_message(VERBOSE, "Spawn scontainer stage 1\n");

        /*
         *  stage1 is responsible for singularity configuration file parsing, handle user input,
         *  read capabilities, check what namespaces is required.
         */
        if ( config.isSuid || geteuid() == 0 ) {
            priv_escalate();
            prepare_scontainer_stage(SCONTAINER_STAGE1);
        }

        return;
    } else if ( stage_pid < 0 ) {
        singularity_message(ERROR, "Failed to spawn scontainer stage 1\n");
        exit(1);
    }

    close(output[1]);

    fds[0].fd = output[0];
    fds[0].events = POLLIN;
    fds[0].revents = 0;

    fds[1].fd = master_socket[0];
    fds[1].events = POLLIN;
    fds[1].revents = 0;

    singularity_message(DEBUG, "Wait C and JSON runtime configuration from scontainer stage 1\n");

    while ( poll(fds, 2, -1) >= 0 ) {
        if ( fds[0].revents & POLLIN ) {
            int ret;
            singularity_message(DEBUG, "Receiving configuration from scontainer stage 1\n");
            if ( (ret = read(fds[0].fd, &config, sizeof(config))) != sizeof(config) ) {
                singularity_message(ERROR, "Failed to read C configuration stdout pipe: %s\n", strerror(errno));
                exit(1);
            }
            if ( config.jsonConfSize >= MAX_JSON_SIZE) {
                singularity_message(ERROR, "JSON configuration too big\n");
                exit(1);
            }
            if ( (ret = read(fds[0].fd, json_stdin, config.jsonConfSize)) != config.jsonConfSize ) {
                singularity_message(ERROR, "Failed to read JSON configuration from stdout pipe: %s\n", strerror(errno));
                exit(1);
            }
            json_stdin[config.jsonConfSize] = '\0';
            break;
        }
        /* TODO: pass file descriptors from stage 1 over unix socket */
        if ( fds[1].revents & POLLIN ) {
            continue;
        }
    }

    close(output[0]);

    singularity_message(DEBUG, "Wait completion of scontainer stage1\n");
    if ( wait(&status) != stage_pid ) {
        singularity_message(ERROR, "Can't wait child\n");
        exit(1);
    }

    if ( WIFEXITED(status) || WIFSIGNALED(status) ) {
        if ( WEXITSTATUS(status) != 0 ) {
            singularity_message(ERROR, "Child exit with status %d\n", WEXITSTATUS(status));
            exit(WEXITSTATUS(status));
        }
    }

    if ( config.isInstance ) {
        singularity_message(VERBOSE, "Run as instance\n");
        int forked = fork();
        if ( forked == 0 ) {
            if ( chdir("/") < 0 ) {
                singularity_message(ERROR, "Can't change directory to /: %s\n", strerror(errno));
                exit(1);
            }
            if ( setsid() < 0 ) {
                singularity_message(ERROR, "Can't set session leader: %s\n", strerror(errno));
                exit(1);
            }
            umask(0);
        } else {
            sigset_t usrmask;
            static struct sigaction action;

            action.sa_sigaction = (void *)&do_exit;
            action.sa_flags = SA_SIGINFO|SA_RESTART;

            sigemptyset(&usrmask);
            sigaddset(&usrmask, SIGUSR1);
            sigaddset(&usrmask, SIGUSR2);

            if (sigprocmask(SIG_SETMASK, &usrmask, NULL) == -1) {
                singularity_message(ERROR, "Blocked signals error: %s\n", strerror(errno));
                exit(1);
            }
            if (sigaction(SIGUSR2, &action, NULL) < 0) {
		        singularity_message(ERROR, "Failed to install signal handler for SIGUSR2\n");
                exit(1);
            }
            if (sigaction(SIGUSR1, &action, NULL) < 0) {
		        singularity_message(ERROR, "Failed to install signal handler for SIGUSR1\n");
                exit(1);
            }
            if (sigprocmask(SIG_UNBLOCK, &usrmask, NULL) == -1) {
                singularity_message(ERROR, "Unblock signals error: %s\n", strerror(errno));
                exit(1);
            }
            pause();
        }
    }

    if ( (config.nsFlags & CLONE_NEWUSER) == 0 ) {
        priv_escalate();
    } else {
        if ( config.isSuid ) {
            singularity_message(ERROR, "Running setuid workflow with user namespace is not allowed\n");
            exit(1);
        }
        if ( config.userPid ) {
            if ( enter_namespace(config.userPid, CLONE_NEWUSER) < 0 ) {
                singularity_message(ERROR, "Failed to enter in user namespace: %s\n", strerror(errno));
                exit(1);
            }
        } else {
            setup_userns(&config.uidMapping[0], &config.gidMapping[0]);
        }
    }

    if ( config.mntPid ) {
        if ( enter_namespace(config.mntPid, CLONE_NEWNS) < 0 ) {
            singularity_message(ERROR, "Failed to enter in mount namespace: %s\n", strerror(errno));
            exit(1);
        }
    } else {
        if ( unshare(CLONE_FS) < 0 ) {
            singularity_message(ERROR, "Failed to unshare root file system: %s\n", strerror(errno));
            exit(1);
        }
        if ( create_namespace(CLONE_NEWNS) < 0 ) {
            singularity_message(ERROR, "Failed to create mount namespace: %s\n", strerror(errno));
            exit(1);
        }
        if ( mount(NULL, "/", NULL, MS_PRIVATE|MS_REC, NULL) < 0 ) {
            singularity_message(ERROR, "Failed to propagate as SHARED: %s\n", strerror(errno));
        }
        if ( create_namespace(CLONE_NEWNS) < 0 ) {
            singularity_message(ERROR, "Failed to create mount namespace: %s\n", strerror(errno));
            exit(1);
        }
        if ( mount(NULL, "/", NULL, MS_SHARED|MS_REC, NULL) < 0 ) {
            singularity_message(ERROR, "Failed to propagate as SHARED: %s\n", strerror(errno));
        }
        /* sync smaster and near child with an eventfd */
        syncfd = eventfd(0, 0);
        if ( syncfd < 0 ) {
            singularity_message(ERROR, "Failed to create sync pipe between smaster and child: %s\n", strerror(errno));
            exit(1);
        }
    }
    singularity_message(DEBUG, "Create RPC socketpair for communication between scontainer and RPC server\n");
    if ( socketpair(AF_UNIX, SOCK_STREAM|SOCK_CLOEXEC, 0, rpc_socket) < 0 ) {
        singularity_message(ERROR, "Failed to create communication socket: %s\n", strerror(errno));
        exit(1);
    }
    /* Use setfsuid to address issue about root_squash filesystems option */
    if ( config.isSuid ) {
        if ( setfsuid(uid) != 0 ) {
            singularity_message(ERROR, "Previous filesystem UID is not equal to 0\n");
            exit(1);
        }
        if ( setfsuid(-1) != uid ) {
            singularity_message(ERROR, "Failed to set filesystem uid to %d\n", uid);
            exit(1);
        }
    }
    if ( config.pidPid ) {
        if ( enter_namespace(config.pidPid, CLONE_NEWPID) < 0 ) {
            singularity_message(ERROR, "Failed to enter in pid namespace: %s\n", strerror(errno));
            exit(1);
        }
        stage_pid = fork();
    } else {
        if ( config.nsFlags & CLONE_NEWPID ) {
            singularity_message(VERBOSE, "Create pid namespace\n");
            stage_pid = fork_ns(CLONE_NEWPID);
        } else {
            stage_pid = fork();
        }
    }

    if ( stage_pid == 0 ) {
        /* at this stage we are PID 1 if PID namespace requested */

        set_parent_death_signal(SIGKILL);

        close(master_socket[0]);

        singularity_message(VERBOSE, "Spawn scontainer stage 2\n");

        if ( config.netPid ) {
            if ( enter_namespace(config.netPid, CLONE_NEWNET) < 0 ) {
                singularity_message(ERROR, "Failed to enter in network namespace: %s\n", strerror(errno));
                exit(1);
            }
        } else {
            if ( config.nsFlags & CLONE_NEWNET ) {
                if ( create_namespace(CLONE_NEWNET) < 0 ) {
                    singularity_message(ERROR, "Failed to create network namespace: %s\n", strerror(errno));
                    exit(1);
                }
            }
        }
        if ( config.mntPid == 0 ) {
            if ( create_namespace(CLONE_NEWNS) < 0 ) {
                singularity_message(ERROR, "Failed to create mount namespace: %s\n", strerror(errno));
                exit(1);
            }

            if ( mount(NULL, "/", NULL, MS_SHARED|MS_REC, NULL) < 0 ) {
                singularity_message(ERROR, "Failed to propagate as SHARED: %s\n", strerror(errno));
            }

            if ( syncfd >= 0 ) {
                unsigned long long counter = 1;
                if ( write(syncfd, &counter, sizeof(counter)) != sizeof(counter) ) {
                    singularity_message(ERROR, "Failed to synchronize with smaster: %s\n", strerror(errno));
                    exit(1);
                }
                close(syncfd);
            }
        }
        if ( config.utsPid ) {
            if ( enter_namespace(config.utsPid, CLONE_NEWUTS) < 0 ) {
                singularity_message(ERROR, "Failed to enter in uts namespace: %s\n", strerror(errno));
                exit(1);
            }
        } else {
            if ( config.nsFlags & CLONE_NEWUTS ) {
                if ( create_namespace(CLONE_NEWUTS) < 0 ) {
                    singularity_message(ERROR, "Failed to create uts namespace: %s\n", strerror(errno));
                    exit(1);
                }
            }
        }
        if ( config.ipcPid ) {
            if ( enter_namespace(config.ipcPid, CLONE_NEWIPC) < 0 ) {
                singularity_message(ERROR, "Failed to enter in ipc namespace: %s\n", strerror(errno));
                exit(1);
            }
        } else {
            if ( config.nsFlags & CLONE_NEWIPC ) {
                if ( create_namespace(CLONE_NEWIPC) < 0 ) {
                    singularity_message(ERROR, "Failed to create ipc namespace: %s\n", strerror(errno));
                    exit(1);
                }
            }
        }
        if ( config.cgroupPid ) {
            if ( enter_namespace(config.cgroupPid, CLONE_NEWCGROUP) < 0 ) {
                singularity_message(ERROR, "Failed to enter in cgroup namespace: %s\n", strerror(errno));
                exit(1);
            }
        } else {
            if ( config.nsFlags & CLONE_NEWCGROUP ) {
                if ( create_namespace(CLONE_NEWCGROUP) < 0 ) {
                    singularity_message(ERROR, "Failed to create cgroup namespace: %s\n", strerror(errno));
                    exit(1);
                }
            }
        }

        close(rpc_socket[0]);

        if ( config.mntPid == 0 ) {
            singularity_message(VERBOSE, "Spawn RPC server\n");
            execute = RPC_SERVER;
        } else {
            singularity_message(VERBOSE, "Don't execute RPC server, joining instance\n");
            prepare_scontainer_stage(SCONTAINER_STAGE2);
        }
        return;
    } else if ( stage_pid > 0 ) {
        config.containerPid = stage_pid;

        singularity_message(VERBOSE, "Spawn smaster process\n");

        close(master_socket[1]);
        close(rpc_socket[1]);

        if ( syncfd >= 0 ) {
            unsigned long long counter;

            if ( read(syncfd, &counter, sizeof(counter)) != sizeof(counter) ) {
                singularity_message(ERROR, "Failed to receive sync signal from child: %s\n", strerror(errno));
                exit(1);
            }

            close(syncfd);
        }

        if ( config.mntPid != 0 ) {
            if ( config.isSuid && setresuid(uid, uid, uid) < 0 ) {
                singularity_message(ERROR, "Failed to drop privileges permanently\n");
                exit(1);
            }
            singularity_message(DEBUG, "Wait scontainer stage 2 child process\n");
            waitpid(stage_pid, &status, 0);
            if ( WIFEXITED(status) || WIFSIGNALED(status) ) {
                singularity_message(VERBOSE, "scontainer stage 2 exited with status %d\n", WEXITSTATUS(status));
                exit(WEXITSTATUS(status));
            }
            singularity_message(ERROR, "Child exit with unknown status\n");
            exit(1);
        } else {
            if ( config.isSuid && setresuid(uid, uid, 0) < 0 ) {
                singularity_message(ERROR, "Failed to drop privileges\n");
                exit(1);
            }
            execute = SMASTER;
            return;
        }
    }
    singularity_message(ERROR, "Failed to create container namespaces\n");
    exit(1);
}
