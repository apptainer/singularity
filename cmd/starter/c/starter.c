/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE.md file distributed with the sources of this project regarding
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
#include <dirent.h>
#include <sys/signalfd.h>
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
#include <net/if.h>
#include <sys/eventfd.h>

#ifdef SINGULARITY_SECUREBITS
#  include <linux/securebits.h>
#else
#  include "util/securebits.h"
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

#include "util/capability.h"
#include "util/message.h"

#include "starter.h"

#define CLONE_STACK_SIZE    1024*1024
#define BUFSIZE             512

extern char **environ;

/* C and JSON configuration */
struct cConfig config;
char *json_stdin;
char *nspath;

#define get_nspath(nstype) (config.nstype##NsPathOffset == 0 ? NULL : &nspath[config.nstype##NsPathOffset])

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

    if ( !(config.nsFlags & CLONE_NEWUSER) ) {
        /* apply target UID/GID for root user */
        if ( uid == 0 ) {
            if ( config.numGID != 0 || config.targetUID != 0 ) {
                if ( prctl(PR_SET_SECUREBITS, SECBIT_NO_SETUID_FIXUP|SECBIT_NO_SETUID_FIXUP_LOCKED) < 0 ) {
                    singularity_message(ERROR, "Failed to set securebits: %s\n", strerror(errno));
                    exit(1);
                }
            }

            if ( config.numGID != 0 ) {
                singularity_message(DEBUG, "Clear additional group IDs\n");

                if ( setgroups(0, NULL) < 0 ) {
                    singularity_message(ERROR, "Unabled to clear additional group IDs: %s\n", strerror(errno));
                    exit(1);
                }
            }

            if ( config.numGID >= 2 ) {
                singularity_message(DEBUG, "Set additional group IDs\n");

                if ( setgroups(config.numGID-1, &config.targetGID[1]) < 0 ) {
                    singularity_message(ERROR, "Failed to set additional groups: %s\n", strerror(errno));
                    exit(1);
                }
            }
            if ( config.numGID >= 1 ) {
                gid_t targetGID = config.targetGID[0];

                singularity_message(DEBUG, "Set main group ID\n");

                if ( setresgid(targetGID, targetGID, targetGID) < 0 ) {
                    singularity_message(ERROR, "Failed to set GID %d: %s\n", targetGID, strerror(errno));
                    exit(1);
                }
            }
            if ( config.targetUID != 0 ) {
                uid_t targetUID = config.targetUID;

                singularity_message(DEBUG, "Set user ID to %d\n", targetUID);

                if ( setresuid(targetUID, targetUID, targetUID) < 0 ) {
                    singularity_message(ERROR, "Faile to drop privileges: %s\n", strerror(errno));
                    exit(1);
                }
            }
        } else if ( config.isSuid ) {
            if ( prctl(PR_SET_SECUREBITS, SECBIT_NO_SETUID_FIXUP|SECBIT_NO_SETUID_FIXUP_LOCKED) < 0 ) {
                singularity_message(ERROR, "Failed to set securebits: %s\n", strerror(errno));
                exit(1);
            }

            if ( setresuid(uid, uid, uid) < 0 ) {
                singularity_message(ERROR, "Faile to drop privileges: %s\n", strerror(errno));
                exit(1);
            }
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

static int enter_namespace(char *nspath, int nstype) {
    int ns_fd;

    switch(nstype) {
    case CLONE_NEWPID:
        singularity_message(VERBOSE, "Entering in pid namespace\n");
#ifndef NS_CLONE_NEWPID
        errno = EINVAL;
        return(-1);
#endif /* NS_CLONE_NEWPID */
        break;
    case CLONE_NEWNET:
        singularity_message(VERBOSE, "Entering in network namespace\n");
#ifndef NS_CLONE_NEWNET
        errno = EINVAL;
        return(-1);
#endif /* NS_CLONE_NEWNET */
        break;
    case CLONE_NEWIPC:
        singularity_message(VERBOSE, "Entering in ipc namespace\n");
#ifndef NS_CLONE_NEWIPC
        errno = EINVAL;
        return(-1);
#endif /* NS_CLONE_NEWIPC */
        break;
    case CLONE_NEWNS:
        singularity_message(VERBOSE, "Entering in mount namespace\n");
#ifndef NS_CLONE_NEWNS
        errno = EINVAL;
        return(-1);
#endif /* NS_CLONE_NEWNS */
        break;
    case CLONE_NEWUTS:
        singularity_message(VERBOSE, "Entering in uts namespace\n");
#ifndef NS_CLONE_NEWUTS
        errno = EINVAL;
        return(-1);
#endif /* NS_CLONE_NEWUTS */
        break;
    case CLONE_NEWUSER:
        singularity_message(VERBOSE, "Entering in user namespace\n");
#ifndef NS_CLONE_NEWUSER
        errno = EINVAL;
        return(-1);
#endif /* NS_CLONE_NEWUSER */
        break;
#ifdef NS_CLONE_NEWCGROUP
    case CLONE_NEWCGROUP:
        singularity_message(VERBOSE, "Entering in cgroup namespace\n");
        break;
#endif /* NS_CLONE_NEWCGROUP */
    default:
        singularity_message(VERBOSE, "Entering in unknown namespace\n");
        errno = EINVAL;
        return(-1);
    }

    singularity_message(DEBUG, "Opening namespace file descriptor %s\n", nspath);
    ns_fd = open(nspath, O_RDONLY);
    if ( ns_fd < 0 ) {
        return(-1);
    }

    if ( setns(ns_fd, nstype) < 0 ) {
        int err = errno;
        close(ns_fd);
        errno = err;
        return(-1);
    }

    close(ns_fd);
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

static void list_fd(struct fdlist *fl) {
    int i = 0;
    int fd_proc;
    DIR *dir;
    struct dirent *dirent;

    if ( ( fd_proc = open("/proc/self/fd", O_RDONLY) ) < 0 ) {
        singularity_message(ERROR, "Failed to open /proc/self/fd: %s\n", strerror(errno));
        exit(1);
    }

    if ( ( dir = fdopendir(fd_proc) ) == NULL ) {
        singularity_message(ERROR, "Failed to list /proc/self/fd directory: %s\n", strerror(errno));
        exit(1);
    }

    while ( ( dirent = readdir(dir ) ) ) {
        if ( strcmp(dirent->d_name, ".") == 0 || strcmp(dirent->d_name, "..") == 0 ) {
            continue;
        }
        if ( atoi(dirent->d_name) == fd_proc ) {
            continue;
        }
        fl->num++;
    }

    rewinddir(dir);

    fl->fds = (int *)malloc(sizeof(int)*fl->num);
    if ( fl->fds == NULL ) {
        singularity_message(ERROR, "Memory allocation failed: %s\n", strerror(errno));
        exit(1);
    }

    while ( ( dirent = readdir(dir ) ) ) {
        int cv;
        if ( strcmp(dirent->d_name, ".") == 0 || strcmp(dirent->d_name, "..") == 0 ) {
            continue;
        }

        cv = atoi(dirent->d_name);
        if ( cv == fd_proc ) {
            continue;
        }

        fl->fds[i++] = atoi(dirent->d_name);
    }

    closedir(dir);
    close(fd_proc);
}

static void fix_fsuid(uid_t uid) {
    setfsuid(uid);

    if ( setfsuid(uid) != uid ) {
        singularity_message(ERROR, "Failed to set filesystem uid to %d\n", uid);
        exit(1);
    }
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
    int status;
    struct pollfd fds[2];
    int syncfd = -1;
    int pipe_fd = -1;
    int sfd;
    int i, j;
    struct fdlist fd_before = {NULL, 0};
    struct fdlist fd_after = {NULL, 0};
    char *source, *target;

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

    list_fd(&fd_before);

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

    /* poll on SIGCHLD signal to exit properly if scontainer exit without returning configuration */
    sfd = signalfd(-1, &mask, 0);
    if (sfd == -1) {
        singularity_message(ERROR, "Signalfd failed: %s\n", strerror(errno));
        exit(1);
    }

    singularity_message(DEBUG, "Create socketpair for smaster communication channel\n");
    if ( socketpair(AF_UNIX, SOCK_STREAM|SOCK_CLOEXEC, 0, master_socket) < 0 ) {
        singularity_message(ERROR, "Failed to create communication socket: %s\n", strerror(errno));
        exit(1);
    }

    /*
     *  use CLONE_FILES to share file descriptors opened during stage 1,
     *  this is a lazy implementation to avoid passing file descriptors
     *  between wrapper and stage 1 over unix socket.
     *  This is required so that all processes works with same files/directories
     *  to minimize race conditions
     */
    stage_pid = fork_ns(CLONE_FILES|CLONE_FS);
    if ( stage_pid == 0 ) {
        set_parent_death_signal(SIGKILL);

        singularity_message(VERBOSE, "Spawn scontainer stage 1\n");

        /*
         *  stage1 is responsible for singularity configuration file parsing, handle user input,
         *  read capabilities, check what namespaces is required.
         */
        if ( config.isSuid ) {
            priv_escalate();
            prepare_scontainer_stage(SCONTAINER_STAGE1);
        }

        return;
    } else if ( stage_pid < 0 ) {
        singularity_message(ERROR, "Failed to spawn scontainer stage 1\n");
        exit(1);
    }

    fds[0].fd = master_socket[0];
    fds[0].events = POLLIN;
    fds[0].revents = 0;

    fds[1].fd = sfd;
    fds[1].events = POLLIN;
    fds[1].revents = 0;

    singularity_message(DEBUG, "Wait C and JSON runtime configuration from scontainer stage 1\n");

    while ( poll(fds, 2, -1) >= 0 ) {
        if ( fds[0].revents & POLLIN ) {
            int ret;
            singularity_message(DEBUG, "Receiving configuration from scontainer stage 1\n");
            if ( (ret = read(fds[0].fd, &config, sizeof(config))) != sizeof(config) ) {
                singularity_message(ERROR, "Failed to read C configuration socket: %s\n", strerror(errno));
                exit(1);
            }
            if ( config.nsPathSize >= MAX_NSPATH_SIZE ) {
                singularity_message(ERROR, "Namespace path too long > %d", MAX_NSPATH_SIZE);
                exit(1);
            }
            nspath = (char *)malloc(MAX_NSPATH_SIZE);
            if ( nspath == NULL ) {
                singularity_message(ERROR, "Memory allocation failed: %s\n", strerror(errno));
                exit(1);
            }
            if ( (ret = read(fds[0].fd, nspath, config.nsPathSize)) != config.nsPathSize ) {
                singularity_message(ERROR, "Failed to read namespace path from socket: %s\n", strerror(errno));
                exit(1);
            }
            if ( config.jsonConfSize >= MAX_JSON_SIZE ) {
                singularity_message(ERROR, "JSON configuration too big\n");
                exit(1);
            }
            if ( (ret = read(fds[0].fd, json_stdin, config.jsonConfSize)) != config.jsonConfSize ) {
                singularity_message(ERROR, "Failed to read JSON configuration from socket: %s\n", strerror(errno));
                exit(1);
            }
            json_stdin[config.jsonConfSize] = '\0';
            break;
        }
        if ( fds[1].revents & POLLIN ) {
            break;
        }
    }

    close(sfd);

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

            close(master_socket[0]);
            close(master_socket[1]);

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

    /* relinquish CPU to apply current directory change for current thread */
    sched_yield();

    if ( (config.nsFlags & CLONE_NEWUSER) == 0 && get_nspath(user) == NULL ) {
        priv_escalate();
    } else {
        if ( config.isSuid ) {
            singularity_message(ERROR, "Running setuid workflow with user namespace is not allowed\n");
            exit(1);
        }
        if ( get_nspath(user) ) {
            if ( enter_namespace(get_nspath(user), CLONE_NEWUSER) < 0 ) {
                singularity_message(ERROR, "Failed to enter in user namespace: %s\n", strerror(errno));
                exit(1);
            }
        } else {
            setup_userns(&config.uidMapping[0], &config.gidMapping[0]);
        }
    }

    list_fd(&fd_after);

    source = (char *)malloc(PATH_MAX);
    target = (char *)malloc(PATH_MAX);

    if ( source == NULL || target == NULL ) {
        singularity_message(ERROR, "Memory allocation failed: %s", strerror(errno));
        exit(1);
    }

    /*
     *  close unattended file descriptors opened during scontainer stage 1
     *  execution, that may not be accurate depending of fs operations done
     *  in stage 1, but should work for most engines.
     */
    for ( i = 0; i < fd_after.num; i++ ) {
        struct stat st;
        int found;

        if ( fd_after.fds[i] == master_socket[0] || fd_after.fds[i] == master_socket[1] ) {
            continue;
        }

        found = 0;
        for ( j = 0; j < fd_before.num; j++ ) {
            if ( fd_before.fds[j] == pipe_fd ) {
                continue;
            }
            if ( fd_before.fds[j] == fd_after.fds[i] ) {
                found = 1;
                break;
            }
        }
        if ( found == 1 ) {
            continue;
        }

        memset(target, 0, PATH_MAX);
        snprintf(source, PATH_MAX, "/proc/self/fd/%d", fd_after.fds[i]);

        /* fd with link generating error are closed */
        if ( readlink(source, target, PATH_MAX) < 0 ) {
            close(fd_after.fds[i]);
            continue;
        }
        /* fd pointing to /dev/tty or anonymous inodes are closed */
        if ( strcmp(target, "/dev/tty") == 0 || stat(target, &st) < 0 ) {
            close(fd_after.fds[i]);
            continue;
        }
        /* set force close on exec for remaining fd */
        if ( fcntl(fd_after.fds[i], F_SETFD, FD_CLOEXEC) < 0 ) {
            singularity_message(DEBUG, "Can't set FD_CLOEXEC on file descriptor %d: %s", fd_after.fds[i], strerror(errno));
        }
    }

    free(source);
    free(target);

    if ( get_nspath(mnt) == NULL ) {
        unsigned long propagation = config.mountPropagation;

        if ( propagation == 0 ) {
            propagation = MS_PRIVATE | MS_REC;
        }
        if ( unshare(CLONE_FS) < 0 ) {
            singularity_message(ERROR, "Failed to unshare root file system: %s\n", strerror(errno));
            exit(1);
        }
        if ( create_namespace(CLONE_NEWNS) < 0 ) {
            singularity_message(ERROR, "Failed to create mount namespace: %s\n", strerror(errno));
            exit(1);
        }
        if ( mount(NULL, "/", NULL, propagation, NULL) < 0 ) {
            singularity_message(ERROR, "Failed to set mount propagation: %s\n", strerror(errno));
            exit(1);
        }
        /* set shared mount propagation to share mount points between smaster and container process */
        if ( mount(NULL, "/", NULL, MS_SHARED|MS_REC, NULL) < 0 ) {
            singularity_message(ERROR, "Failed to propagate as SHARED: %s\n", strerror(errno));
            exit(1);
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
        fix_fsuid(uid);
    }
    if ( get_nspath(pid) ) {
        if ( enter_namespace(get_nspath(pid), CLONE_NEWPID) < 0 ) {
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

        if ( get_nspath(net) ) {
            if ( enter_namespace(get_nspath(net), CLONE_NEWNET) < 0 ) {
                singularity_message(ERROR, "Failed to enter in network namespace: %s\n", strerror(errno));
                exit(1);
            }
        } else {
            if ( config.nsFlags & CLONE_NEWNET ) {
                if ( create_namespace(CLONE_NEWNET) < 0 ) {
                    singularity_message(ERROR, "Failed to create network namespace: %s\n", strerror(errno));
                    exit(1);
                }

                struct ifreq req;
                int sockfd = socket(AF_INET, SOCK_DGRAM, 0);

                if ( sockfd < 0 ) {
                    singularity_message(ERROR, "Unable to open AF_INET socket: %s\n", strerror(errno));
                    exit(1);
                }

                memset(&req, 0, sizeof(req));
                strncpy(req.ifr_name, "lo", IFNAMSIZ);

                req.ifr_flags |= IFF_UP;

                singularity_message(DEBUG, "Bringing up network loopback interface\n");
                if ( ioctl(sockfd, SIOCSIFFLAGS, &req) < 0 ) {
                    singularity_message(ERROR, "Failed to set flags on interface: %s\n", strerror(errno));
                    exit(1);
                }
                close(sockfd);
            }
        }
        if ( get_nspath(uts) ) {
            if ( enter_namespace(get_nspath(uts), CLONE_NEWUTS) < 0 ) {
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
        if ( get_nspath(ipc) ) {
            if ( enter_namespace(get_nspath(ipc), CLONE_NEWIPC) < 0 ) {
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
        if ( get_nspath(cgroup) ) {
            if ( enter_namespace(get_nspath(cgroup), CLONE_NEWCGROUP) < 0 ) {
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
        if ( get_nspath(mnt) == NULL ) {
            /* create a namespace for container process to separate smaster during pivot_root */
            if ( create_namespace(CLONE_NEWNS) < 0 ) {
                singularity_message(ERROR, "Failed to create mount namespace: %s\n", strerror(errno));
                exit(1);
            }

            /* set shared propagation to propagate few mount points to smaster */
            if ( mount(NULL, "/", NULL, MS_SHARED|MS_REC, NULL) < 0 ) {
                singularity_message(ERROR, "Failed to propagate as SHARED: %s\n", strerror(errno));
                exit(1);
            }

            if ( syncfd >= 0 ) {
                unsigned long long counter = 1;
                if ( write(syncfd, &counter, sizeof(counter)) != sizeof(counter) ) {
                    singularity_message(ERROR, "Failed to synchronize with smaster: %s\n", strerror(errno));
                    exit(1);
                }
                close(syncfd);
            }
        } else {
            if ( enter_namespace(get_nspath(mnt), CLONE_NEWNS) < 0 ) {
                singularity_message(ERROR, "Failed to enter in mount namespace: %s\n", strerror(errno));
                exit(1);
            }
        }

        close(rpc_socket[0]);

        if ( get_nspath(mnt) == NULL ) {
            /*
             * fork is a convenient way to apply capabilities and privileges drop
             * from single thread context before entering in stage 2
             */
            int process = fork_ns(CLONE_FS|CLONE_FILES);

            if ( process == 0 ) {
                singularity_message(VERBOSE, "Spawn RPC server\n");
                execute = RPC_SERVER;
            } else if ( process > 0 ) {
                int status;

                if ( wait(&status) != process ) {
                    singularity_message(ERROR, "Error while waiting RPC server: %s\n", strerror(errno));
                    exit(1);
                }

                prepare_scontainer_stage(SCONTAINER_STAGE2);
                execute = SCONTAINER_STAGE2;
            } else {
                singularity_message(ERROR, "fork failed: %s\n", strerror(errno));
                exit(1);
            }
        } else {
            singularity_message(VERBOSE, "Don't execute RPC server, joining instance\n");
            prepare_scontainer_stage(SCONTAINER_STAGE2);
        }
        return;
    } else if ( stage_pid > 0 ) {
        pid_t parent_pgrp = getpgid(getppid());
        pid_t tcpgrp = tcgetpgrp(STDIN_FILENO);
        pid_t pgrp = getpgrp();

        if ( tcpgrp == pgrp && parent_pgrp != pgrp ) {
            singularity_message(DEBUG, "Pass terminal control to child\n");

            if ( setpgid(stage_pid, stage_pid) < 0 ) {
                singularity_message(ERROR, "Failed to set child process group: %s\n", strerror(errno));
                exit(1);
            }
            if ( tcsetpgrp(STDIN_FILENO, stage_pid) < 0 ) {
                singularity_message(ERROR, "Failed to set child as foreground process: %s\n", strerror(errno));
                exit(1);
            }
        }

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

        if ( get_nspath(mnt) ) {
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
