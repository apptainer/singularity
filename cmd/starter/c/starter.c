/*
  Copyright (c) 2018-2019, Sylabs, Inc. All rights reserved.

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
#include <dirent.h>
#include <libgen.h>
#include <limits.h>
#include <sys/mman.h>
#include <sys/fsuid.h>
#include <sys/mount.h>
#include <sys/wait.h>
#include <sys/prctl.h>
#include <sys/socket.h>
#include <sys/stat.h>
#include <signal.h>
#include <sched.h>
#include <setjmp.h>
#include <sys/syscall.h>
#include <net/if.h>
#include <sys/eventfd.h>

#ifdef SINGULARITY_SECUREBITS
#  include <linux/securebits.h>
#else
#  include "include/securebits.h"
#endif /* SINGULARITY_SECUREBITS */

#include "include/capability.h"
#include "include/message.h"
#include "include/starter.h"

#define SELF_PID_NS     "/proc/self/ns/pid"
#define SELF_NET_NS     "/proc/self/ns/net"
#define SELF_UTS_NS     "/proc/self/ns/uts"
#define SELF_IPC_NS     "/proc/self/ns/ipc"
#define SELF_MNT_NS     "/proc/self/ns/mnt"
#define SELF_CGROUP_NS  "/proc/self/ns/cgroup"

/* current starter configuration */
struct starterConfig *sconfig;

/* Socket process communication */
int rpc_socket[2] = {-1, -1};
int master_socket[2] = {-1, -1};

/* set Go execution call after init function returns */
enum goexec goexecute;

typedef struct fdlist {
    int *fds;
    unsigned int num;
} fdlist_t;

typedef struct stack {
    char alloc[4096] __attribute__((aligned(16)));
    char ptr[0];
} fork_stack_t;

/* child function called by clone to return directly to sigsetjmp in fork_ns */
__attribute__((noinline)) static int clone_fn(void *arg) {
    siglongjmp(*(sigjmp_buf *)arg, 0);
}

__attribute__ ((returns_twice)) __attribute__((noinline)) static int fork_ns(unsigned int flags) {
    /* setup the stack */
    fork_stack_t stack;
    sigjmp_buf env;

    /*
     * sigsetjmp return 0 when called directly, and will return 1
     * after siglongjmp call in clone_fn. We always save signal mask.
     */
    if ( sigsetjmp(env, 1) ) {
        /* child process will return here after siglongjmp call in clone_fn */
        return 0;
    }
    /* parent process */
    return clone(clone_fn, stack.ptr, (SIGCHLD|flags), env);
}

static void priv_escalate(bool keep_fsuid) {
    uid_t uid = getuid();

    verbosef("Get root privileges\n");
    if ( seteuid(0) < 0 ) {
        fatalf("Failed to set effective UID to 0\n");
    }

    if ( keep_fsuid ) {
        /* Use setfsuid to address issue about root_squash filesystems option */
        verbosef("Change filesystem uid to %d\n", uid);
        setfsuid(uid);
        if ( setfsuid(uid) != uid ) {
            fatalf("Failed to set filesystem uid to %d\n", uid);
        }
    }
}

static void priv_drop(bool permanent) {
    uid_t uid = getuid();
    gid_t gid = getgid();

    if ( !permanent ) {
        verbosef("Drop root privileges\n");
        if ( setegid(gid) < 0 ) {
            fatalf("Failed to set effective GID to %d\n", gid);
        }
        if ( seteuid(uid) < 0 ) {
            fatalf("Failed to set effective UID to %d\n", uid);
        }
    } else {
        verbosef("Drop root privileges permanently\n");
        if ( setresgid(gid, gid, gid) < 0 ) {
            fatalf("Failed to set all GID to %d\n", gid);
        }
        if ( setresuid(uid, uid, uid) < 0 ) {
            fatalf("Failed to set all UID to %d\n", uid);
        }
    }
}

static void set_parent_death_signal(int signo) {
    debugf("Set parent death signal to %d\n", signo);
    if ( prctl(PR_SET_PDEATHSIG, signo) < 0 ) {
        fatalf("Failed to set parent death signal\n");
    }
}

/* helper to check if namespace flag is set */
static inline bool is_namespace_create(struct namespace *nsconfig, unsigned int nsflag) {
    return (nsconfig->flags & nsflag) != 0;
}

/* helper to check if the corresponding namespace need to be joined */
static bool is_namespace_enter(const char *nspath, const char *selfns) {
    if ( selfns != NULL && nspath[0] != 0 ) {
        struct stat selfns_st;
        struct stat ns_st;

        /*
         * errors are logged for debug purpose, and any
         * error implies to not enter in the corresponding
         * namespace, we can safely assume that if an error
         * occurred with those calls, it will also occurred
         * later with open/setns call in enter_namespace
         */
        if ( stat(selfns, &selfns_st) < 0 ) {
            if ( errno != ENOENT ) {
                debugf("Could not stat %s: %s\n", selfns, strerror(errno));
            }
            return false;
        }
        if ( stat(nspath, &ns_st) < 0 ) {
            if ( errno != ENOENT ) {
                debugf("Could not stat %s: %s\n", nspath, strerror(errno));
            }
            return false;
        }
        /* same namespace, don't join */
        if ( selfns_st.st_ino == ns_st.st_ino ) {
            return false;
        }
    }
    return nspath[0] != 0;
}

static int apply_container_privileges(struct privileges *privileges) {
    uid_t currentUID = getuid();
    uid_t targetUID = currentUID;
    struct __user_cap_header_struct header;
    struct __user_cap_data_struct data[2];

    header.version = LINUX_CAPABILITY_VERSION;
    header.pid = 0;

    if ( capget(&header, data) < 0 ) {
        fatalf("Failed to get processus capabilities\n");
    }

    data[1].inheritable = (__u32)(privileges->capabilities.inheritable >> 32);
    data[0].inheritable = (__u32)(privileges->capabilities.inheritable & 0xFFFFFFFF);
    data[1].permitted = (__u32)(privileges->capabilities.permitted >> 32);
    data[0].permitted = (__u32)(privileges->capabilities.permitted & 0xFFFFFFFF);
    data[1].effective = (__u32)(privileges->capabilities.effective >> 32);
    data[0].effective = (__u32)(privileges->capabilities.effective & 0xFFFFFFFF);

    int last_cap;
    for ( last_cap = CAPSET_MAX; ; last_cap-- ) {
        if ( prctl(PR_CAPBSET_READ, last_cap) > 0 || last_cap == 0 ) {
            break;
        }
    }

    int caps_index;
    for ( caps_index = 0; caps_index <= last_cap; caps_index++ ) {
        if ( !(privileges->capabilities.bounding & (1ULL << caps_index)) ) {
            if ( prctl(PR_CAPBSET_DROP, caps_index) < 0 ) {
                fatalf("Failed to drop bounding capabilities set: %s\n", strerror(errno));
            }
        }
    }

    /*
     * prevent capabilities from being adjusted by kernel when changing uid/gid,
     * we need to keep capabilities to apply container capabilities during capset call
     * and to set ambient capabilities. We can't use capset before changing uid/gid
     * because CAP_SETUID/CAP_SETGID could be already dropped
     */
    if ( prctl(PR_SET_SECUREBITS, SECBIT_NO_SETUID_FIXUP|SECBIT_NO_SETUID_FIXUP_LOCKED) < 0 ) {
        fatalf("Failed to set securebits: %s\n", strerror(errno));
    }

    /* apply target GID for root user or if setgroups is allowed within user namespace */
    if ( currentUID == 0 || privileges->allowSetgroups ) {
        if ( privileges->numGID >= 1 ) {
            gid_t targetGID = privileges->targetGID[0];

            debugf("Set main group ID to %d\n", targetGID);
            if ( setresgid(targetGID, targetGID, targetGID) < 0 ) {
                fatalf("Failed to set GID %d: %s\n", targetGID, strerror(errno));
            }

            debugf("Set %d additional group IDs\n", privileges->numGID);
            if ( setgroups(privileges->numGID, privileges->targetGID) < 0 ) {
                fatalf("Failed to set additional groups: %s\n", strerror(errno));
            }
        }
    }
    /* apply target UID for root user, also apply if user namespace UID is zero */
    if ( currentUID == 0 ) {
        targetUID = privileges->targetUID;
    }

    debugf("Set user ID to %d\n", targetUID);
    if ( setresuid(targetUID, targetUID, targetUID) < 0 ) {
        fatalf("Failed to set all user ID to %d: %s\n", targetUID, strerror(errno));
    }

    set_parent_death_signal(SIGKILL);

    if ( privileges->noNewPrivs ) {
        if ( prctl(PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0) < 0 ) {
            fatalf("Failed to set no new privs flag: %s\n", strerror(errno));
        }
        if ( prctl(PR_GET_NO_NEW_PRIVS, 0, 0 ,0, 0) != 1 ) {
            fatalf("Aborting, failed to set no new privs flag: %s\n", strerror(errno));
        }
    }

    if ( capset(&header, data) < 0 ) {
        fatalf("Failed to set process capabilities\n");
    }

#ifdef USER_CAPABILITIES
    // set ambient capabilities if supported
    for ( caps_index = 0; caps_index <= last_cap; caps_index++ ) {
        if ( (privileges->capabilities.ambient & (1ULL << caps_index)) ) {
            if ( prctl(PR_CAP_AMBIENT, PR_CAP_AMBIENT_RAISE, caps_index, 0, 0) < 0 ) {
                fatalf("Failed to set ambient capability: %s\n", strerror(errno));
            }
        }
    }
#endif
}

static int create_namespace(int nstype) {
    switch(nstype) {
    case CLONE_NEWNET:
        verbosef("Create network namespace\n");
        break;
    case CLONE_NEWIPC:
        verbosef("Create ipc namespace\n");
        break;
    case CLONE_NEWNS:
        verbosef("Create mount namespace\n");
        break;
    case CLONE_NEWUTS:
        verbosef("Create uts namespace\n");
        break;
    case CLONE_NEWUSER:
        verbosef("Create user namespace\n");
        break;
    case CLONE_NEWCGROUP:
        verbosef("Create cgroup namespace\n");
        break;
    default:
        warningf("Skipping unknown namespace creation\n");
        errno = EINVAL;
        return(-1);
    }
    return unshare(nstype);
}

static int enter_namespace(char *nspath, int nstype) {
    int ns_fd;

    switch(nstype) {
    case CLONE_NEWPID:
        verbosef("Entering in pid namespace\n");
        break;
    case CLONE_NEWNET:
        verbosef("Entering in network namespace\n");
        break;
    case CLONE_NEWIPC:
        verbosef("Entering in ipc namespace\n");
        break;
    case CLONE_NEWNS:
        verbosef("Entering in mount namespace\n");
        break;
    case CLONE_NEWUTS:
        verbosef("Entering in uts namespace\n");
        break;
    case CLONE_NEWUSER:
        verbosef("Entering in user namespace\n");
        break;
    case CLONE_NEWCGROUP:
        verbosef("Entering in cgroup namespace\n");
        break;
    default:
        verbosef("Entering in unknown namespace\n");
        errno = EINVAL;
        return(-1);
    }

    debugf("Opening namespace file %s\n", nspath);
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

static void set_mappings_external(const char *name, char *cmdpath, pid_t pid, char *map) {
    int ret;
    char *ptr;
    char *cmd = (char *)malloc(MAX_CMD_SIZE);

    if ( !cmdpath[0] ) {
        fatalf("%s is not installed on your system\n", name);
    }

    if ( cmd == NULL ) {
        fatalf("memory allocation failed: %s", strerror(errno));
    }
    memset(cmd, 0, MAX_CMD_SIZE);

    /* replace newlines by space for command execution */
    ptr = map;
    while ( *ptr != '\0' ) {
        if ( *ptr == '\n' ) {
            *ptr = 0x20;
        }
        ptr++;
    }

    /* prepare command line */
    ret = snprintf(cmd, MAX_CMD_SIZE-1, "%s %d %s>/dev/null", cmdpath, pid, map);
    if ( ret > MAX_CMD_SIZE-1 ) {
        fatalf("%s command line truncated", name);
    }

    /* scary !? it's fine as it's never called by setuid context */
    if ( system(cmd) < 0 ) {
        fatalf("'%s' execution failed", cmd);
    }

    free(cmd);
}

/*
 * write user namespace mapping via external binaries newuidmap
 * and newgidmap. This function is only called by unprivileged
 * installation
 */
static void setup_userns_mappings_external(struct container *container) {
    struct privileges *privileges = &container->privileges;

    set_mappings_external(
        "newgidmap",
        privileges->newgidmapPath,
        container->pid,
        privileges->gidMap
    );
    set_mappings_external(
        "newuidmap",
        privileges->newuidmapPath,
        container->pid,
        privileges->uidMap
    );
}

/*
 * write user namespace mapping, this function must be called
 * after the calling process entered in corresponding /proc/<pid>
 * directory, because it will write setgroups/uid_map/gid_map file
 * relative to the targeted process directory
 */
static void setup_userns_mappings(struct privileges *privileges) {
    FILE *map_fp;
    char *allow = "allow", *deny = "deny";
    char *setgroup = deny;

    if ( privileges->allowSetgroups ) {
        setgroup = allow;
    }

    debugf("Write %s to setgroups file\n", setgroup);
    map_fp = fopen("setgroups", "w+");
    if ( map_fp != NULL ) {
        fprintf(map_fp, "%s\n", setgroup);
        if ( fclose(map_fp) < 0 ) {
            fatalf("Failed to write %s to setgroups file: %s\n", setgroup, strerror(errno));
        }
    } else {
        fatalf("Could not write info to setgroups: %s\n", strerror(errno));
    }

    debugf("Write to GID map\n");
    map_fp = fopen("gid_map", "w+");
    if ( map_fp != NULL ) {
        fprintf(map_fp, "%s", privileges->gidMap);
        if ( fclose(map_fp) < 0 ) {
            fatalf("Failed to write to GID map: %s\n", strerror(errno));
        }
    } else {
        fatalf("Could not write parent info to gid_map: %s\n", strerror(errno));
    }

    debugf("Write to UID map\n");
    map_fp = fopen("uid_map", "w+");
    if ( map_fp != NULL ) {
        fprintf(map_fp, "%s", privileges->uidMap);
        if ( fclose(map_fp) < 0 ) {
            fatalf("Failed to write to UID map: %s\n", strerror(errno));
        }
    } else {
        fatalf("Could not write parent info to uid_map: %s\n", strerror(errno));
    }
}

static int user_namespace_init(struct namespace *nsconfig) {
    if ( is_namespace_enter(nsconfig->user, NULL) ) {
        if ( enter_namespace(nsconfig->user, CLONE_NEWUSER) < 0 ) {
            fatalf("Failed to enter in user namespace: %s\n", strerror(errno));
        }
        return ENTER_NAMESPACE;
    } else if ( is_namespace_create(nsconfig, CLONE_NEWUSER) ) {
        verbosef("Create user namespace\n");
        return CREATE_NAMESPACE;
    }
    return NO_NAMESPACE;
}

static int pid_namespace_init(struct namespace *nsconfig) {
    if ( is_namespace_enter(nsconfig->pid, SELF_PID_NS) ) {
        if ( enter_namespace(nsconfig->pid, CLONE_NEWPID) < 0 ) {
            fatalf("Failed to enter in pid namespace: %s\n", strerror(errno));
        }
        return ENTER_NAMESPACE;
    } else if ( is_namespace_create(nsconfig, CLONE_NEWPID) ) {
        verbosef("Create pid namespace\n");
        return CREATE_NAMESPACE;
    }
    return NO_NAMESPACE;
}

static int network_namespace_init(struct namespace *nsconfig) {
    if ( is_namespace_enter(nsconfig->network, SELF_NET_NS) ) {
        if ( enter_namespace(nsconfig->network, CLONE_NEWNET) < 0 ) {
            fatalf("Failed to enter in network namespace: %s\n", strerror(errno));
        }
        return ENTER_NAMESPACE;
    } else if ( is_namespace_create(nsconfig, CLONE_NEWNET) ) {
        if ( create_namespace(CLONE_NEWNET) < 0 ) {
            fatalf("Failed to create network namespace: %s\n", strerror(errno));
        }
        if ( nsconfig->bringLoopbackInterface ) {
            struct ifreq req;
            int sockfd = socket(AF_INET, SOCK_DGRAM, 0);

            if ( sockfd < 0 ) {
                fatalf("Unable to open AF_INET socket: %s\n", strerror(errno));
            }

            memset(&req, 0, sizeof(req));
            strncpy(req.ifr_name, "lo", IFNAMSIZ);

            req.ifr_flags |= IFF_UP;

            debugf("Bringing up network loopback interface\n");
            if ( ioctl(sockfd, SIOCSIFFLAGS, &req) < 0 ) {
                fatalf("Failed to set flags on interface: %s\n", strerror(errno));
            }
            close(sockfd);
        }
        return CREATE_NAMESPACE;
    }
    return NO_NAMESPACE;
}

static int uts_namespace_init(struct namespace *nsconfig) {
    if ( is_namespace_enter(nsconfig->uts, SELF_UTS_NS) ) {
        if ( enter_namespace(nsconfig->uts, CLONE_NEWUTS) < 0 ) {
            fatalf("Failed to enter in uts namespace: %s\n", strerror(errno));
        }
        return ENTER_NAMESPACE;
    } else if ( is_namespace_create(nsconfig, CLONE_NEWUTS) ) {
        if ( create_namespace(CLONE_NEWUTS) < 0 ) {
            fatalf("Failed to create uts namespace: %s\n", strerror(errno));
        }
        return CREATE_NAMESPACE;
    }
}

static int ipc_namespace_init(struct namespace *nsconfig) {
    if ( is_namespace_enter(nsconfig->ipc, SELF_IPC_NS) ) {
        if ( enter_namespace(nsconfig->ipc, CLONE_NEWIPC) < 0 ) {
            fatalf("Failed to enter in ipc namespace: %s\n", strerror(errno));
        }
        return ENTER_NAMESPACE;
    } else if ( is_namespace_create(nsconfig, CLONE_NEWIPC) ) {
        if ( create_namespace(CLONE_NEWIPC) < 0 ) {
            fatalf("Failed to create ipc namespace: %s\n", strerror(errno));
        }
        return CREATE_NAMESPACE;
    }
}

static int cgroup_namespace_init(struct namespace *nsconfig) {
    if ( is_namespace_enter(nsconfig->cgroup, SELF_CGROUP_NS) ) {
        if ( enter_namespace(nsconfig->cgroup, CLONE_NEWCGROUP) < 0 ) {
            fatalf("Failed to enter in cgroup namespace: %s\n", strerror(errno));
        }
        return ENTER_NAMESPACE;
    } else if ( is_namespace_create(nsconfig, CLONE_NEWCGROUP) ) {
        if ( create_namespace(CLONE_NEWCGROUP) < 0 ) {
            fatalf("Failed to create cgroup namespace: %s\n", strerror(errno));
        }
        return CREATE_NAMESPACE;
    }
}

static int mount_namespace_init(struct namespace *nsconfig, bool masterPropagateMount) {
    if ( is_namespace_enter(nsconfig->mount, SELF_MNT_NS) ) {
        if ( enter_namespace(nsconfig->mount, CLONE_NEWNS) < 0 ) {
            fatalf("Failed to enter in mount namespace: %s\n", strerror(errno));
        }
        return ENTER_NAMESPACE;
    } else if ( is_namespace_create(nsconfig, CLONE_NEWNS) ) {
        if ( !masterPropagateMount ) {
            unsigned long propagation = nsconfig->mountPropagation;

            if ( unshare(CLONE_FS) < 0 ) {
                fatalf("Failed to unshare root file system: %s\n", strerror(errno));
            }
            if ( create_namespace(CLONE_NEWNS) < 0 ) {
                fatalf("Failed to create mount namespace: %s\n", strerror(errno));
            }
            if ( propagation && mount(NULL, "/", NULL, propagation, NULL) < 0 ) {
                fatalf("Failed to set mount propagation: %s\n", strerror(errno));
            }
        } else {
            /* create a namespace for container process to separate master during pivot_root */
            if ( create_namespace(CLONE_NEWNS) < 0 ) {
                fatalf("Failed to create mount namespace: %s\n", strerror(errno));
            }

            /* set shared propagation to propagate few mount points to master */
            if ( mount(NULL, "/", NULL, MS_SHARED|MS_REC, NULL) < 0 ) {
                fatalf("Failed to propagate as SHARED: %s\n", strerror(errno));
            }
        }
        return CREATE_NAMESPACE;
    }
    return NO_NAMESPACE;
}

static int shared_mount_namespace_init(struct namespace *nsconfig) {
    unsigned long propagation = nsconfig->mountPropagation;

    if ( propagation == 0 ) {
        propagation = MS_PRIVATE | MS_REC;
    }
    if ( unshare(CLONE_FS) < 0 ) {
        fatalf("Failed to unshare root file system: %s\n", strerror(errno));
    }
    if ( create_namespace(CLONE_NEWNS) < 0 ) {
        fatalf("Failed to create mount namespace: %s\n", strerror(errno));
    }
    if ( mount(NULL, "/", NULL, propagation, NULL) < 0 ) {
        fatalf("Failed to set mount propagation: %s\n", strerror(errno));
    }
    /* set shared mount propagation to share mount points between master and container process */
    if ( mount(NULL, "/", NULL, MS_SHARED|MS_REC, NULL) < 0 ) {
        fatalf("Failed to propagate as SHARED: %s\n", strerror(errno));
    }
    return CREATE_NAMESPACE;
}

static bool is_suid(void) {
    ElfW(auxv_t) *auxv;
    bool suid = 0;
    char *buffer = (char *)malloc(4096);
    int proc_auxv = open("/proc/self/auxv", O_RDONLY);

    verbosef("Check if we are running as setuid\n");

    if ( proc_auxv < 0 ) {
        fatalf("Can't open /proc/self/auxv: %s\n", strerror(errno));
    }

    /* use auxiliary vectors to determine if running privileged */
    memset(buffer, 0, 4096);
    if ( read(proc_auxv, buffer, 4088) < 0 ) {
        fatalf("Can't read auxiliary vectors: %s\n", strerror(errno));
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

static fdlist_t *list_fd(void) {
    int i = 0;
    int fd_proc;
    DIR *dir;
    struct dirent *dirent;
    fdlist_t *fl = (fdlist_t *)malloc(sizeof(fdlist_t));

    if ( fl == NULL ) {
        fatalf("Memory allocation failed: %s\n", strerror(errno));
    }

    fl->fds = NULL;
    fl->num = 0;

    if ( ( fd_proc = open("/proc/self/fd", O_RDONLY) ) < 0 ) {
        fatalf("Failed to open /proc/self/fd: %s\n", strerror(errno));
    }

    if ( ( dir = fdopendir(fd_proc) ) == NULL ) {
        fatalf("Failed to list /proc/self/fd directory: %s\n", strerror(errno));
    }

    while ( ( dirent = readdir(dir) ) ) {
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
        fatalf("Memory allocation failed: %s\n", strerror(errno));
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

        fl->fds[i++] = cv;
    }

    closedir(dir);
    close(fd_proc);

    return fl;
}

static void cleanup_fd(fdlist_t *master, struct starter *starter) {
    int fd_proc;
    DIR *dir;
    struct dirent *dirent;
    int i, fd, found;

    if ( ( fd_proc = open("/proc/self/fd", O_RDONLY) ) < 0 ) {
        fatalf("Failed to open /proc/self/fd: %s\n", strerror(errno));
    }

    if ( ( dir = fdopendir(fd_proc) ) == NULL ) {
        fatalf("Failed to list /proc/self/fd directory: %s\n", strerror(errno));
    }

    while ( ( dirent = readdir(dir) ) ) {
        if ( strcmp(dirent->d_name, ".") == 0 || strcmp(dirent->d_name, "..") == 0 ) {
            continue;
        }
        fd = atoi(dirent->d_name);
        if ( fd == fd_proc ) {
            continue;
        }

        found = 0;

        /* check if the file descriptor was open before stage 1 execution */
        for ( i = 0; i < master->num; i++ ) {
            if ( master->fds[i] == fd ) {
                found++;
                break;
            }
        }
        if ( found ) {
            continue;
        }

        found = 0;

        /* check if the file descriptor need to remain opened */
        for ( i = 0; i < starter->numfds; i++ ) {
            if ( starter->fds[i] == fd ) {
                found++;
                /* set force close on exec */
                if ( fcntl(starter->fds[i], F_SETFD, FD_CLOEXEC) < 0 ) {
                    debugf("Can't set FD_CLOEXEC on file descriptor %d: %s\n", starter->fds[i], strerror(errno));
                }
                break;
            }
        }

        /* close unattended file descriptors opened during stage 1 execution */
        if ( !found ) {
            debugf("Close file descriptor %d\n", fd);
            close(fd);
        }
    }

    closedir(dir);
    close(fd_proc);
}

static int wait_event(int fd) {
    unsigned char val = 1;
    if ( read(fd, &val, sizeof(unsigned char)) <= 0 ) {
        return(-1);
    }
    return(0);
}

static int send_event(int fd) {
    unsigned char val = 1;
    if ( write(fd, &val, sizeof(unsigned char)) <= 0 ) {
        return(-1);
    }
    return(0);
}

static void chdir_to_proc_pid(pid_t pid) {
    char *buffer = (char *)malloc(128);

    if ( buffer == NULL ) {
        fatalf("memory allocation failed: %s\n", strerror(errno));
    }

    memset(buffer, 0, 128);
    snprintf(buffer, 127, "/proc/%d", pid);

    if ( chdir(buffer) < 0 ) {
        fatalf("Failed to change directory to %s: %s\n", buffer, strerror(errno));
    }

    /* check that process is a child */
    if ( getpgid(0) != getpgid(pid) ) {
        fatalf("Could not change directory to %s: bad process\n", buffer);
    }

    free(buffer);
}

static void fix_streams(void) {
    struct stat st;
    int i = 0;
    int null = open("/dev/null", O_RDONLY);

    if ( null <= 2 ) {
        i = null;
    }

    for ( ; i <= 2; i++ ) {
        if ( fstat(i, &st) < 0 && errno == EBADF ) {
            if ( dup2(null, i) < 0 ) {
                fatalf("Error while fixing IO streams: %s\n", strerror(errno));
            }
        }
    }

    if ( null > 2 ) {
        close(null);
    }
}

static void wait_child(const char *name, pid_t child_pid, bool noreturn) {
    int status;
    int exit_status = 0;

    pid_t pid = waitpid(child_pid, &status, 0);
    if ( pid < 0 ) {
        fatalf("Failed to wait %s: %s\n", name, strerror(errno));
    } else if ( pid != child_pid ) {
        fatalf("Unexpected child (pid %d) status received\n", pid);
    }

    if ( WIFEXITED(status) ) {
        if ( WEXITSTATUS(status) != 0 ) {
            exit_status = WEXITSTATUS(status);
        }
        verbosef("%s exited with status %d\n", name, exit_status);
        /* noreturn will exit the current process with corresponding status */
        if ( noreturn || exit_status != 0 ) {
            exit(exit_status);
        }
    } else if ( WIFSIGNALED(status) ) {
        verbosef("%s interrupted by signal number %d\n", name, WTERMSIG(status));
        kill(getpid(), WTERMSIG(status));
        /* we should never return from kill with signal default actions */
        exit(128 + WTERMSIG(status));
    } else {
        fatalf("%s exited with unknown status\n", name);
    }
}

void do_exit(int sig) {
    exit(0);
}

/*
 * as clearenv set environ pointer to NULL, it only works
 * in C context and doesn't have any effect with the Go
 * runtime using the real pointer, so we need to work
 * directly with environment stack with cleanenv function.
 */
static void cleanenv(void) {
    extern char **environ;
    char **e;

    if ( environ == NULL || *environ == NULL ) {
        fatalf("no environment variables set\n");
    }

    /* 
     * keep only SINGULARITY_MESSAGELEVEL for GO runtime, set others to empty
     * string and not NULL (see issue #3703 for why)
     */
    for (e = environ; *e != NULL; e++) {
        if ( strncmp(MSGLVL_ENV "=", *e, sizeof(MSGLVL_ENV)) != 0 ) {
            *e = "";
        }
    }
}

/*
 * get_pipe_exec_fd returns the pipe file descriptor stored in
 * the PIPE_EXEC_FD environment variable
 */
static int get_pipe_exec_fd(void) {
    int pipe_fd;
    char *pipe_fd_env = getenv("PIPE_EXEC_FD");

    if ( pipe_fd_env != NULL ) {
        if ( sscanf(pipe_fd_env, "%d", &pipe_fd) != 1 ) {
            fatalf("Failed to parse PIPE_EXEC_FD environment variable: %s\n", strerror(errno));
        }
        debugf("PIPE_EXEC_FD value: %d\n", pipe_fd);
        if ( pipe_fd < 0 || pipe_fd >= sysconf(_SC_OPEN_MAX) ) {
            fatalf("Bad PIPE_EXEC_FD file descriptor value\n");
        }
    } else {
        fatalf("PIPE_EXEC_FD environment variable isn't set\n");
    }

    return pipe_fd;
}

/*
* Starter's entrypoint executed before Go runtime in a single-thread context.
*
* The constructor attribute causes init(void) function to be called automatically before
* execution enters main(). This behavior is required in order to prepare isolated environment
* for a container. Init will create and(or) enter requested namespaces delegating setup work
* to the specific engine. Init forks oneself a couple times during execution which allows engine
* to perform initialization inside the container context (RPC server) and outside of it (CreateContainer
* method of an engine). At the end only two processed will be left: a container process in the prepared
* environment and a master process outside of it that monitors container's state.
*/
__attribute__((constructor)) static void init(void) {
    uid_t uid = getuid();
    sigset_t mask;
    pid_t process;
    int pipe_fd = -1;
    int clone_flags = 0;
    int userns = NO_NAMESPACE, pidns = NO_NAMESPACE;
    fdlist_t *master_fds;

    verbosef("Starter initialization\n");

#ifndef SINGULARITY_NO_NEW_PRIVS
    fatalf("Host kernel is outdated and does not support PR_SET_NO_NEW_PRIVS!\n");
#endif

    /*
     * get pipe file descriptor from environment variable PIPE_EXEC_FD
     * to read engine configuration
     */
    pipe_fd = get_pipe_exec_fd();

    /* cleanup environment variables */
    cleanenv();

    /* initialize starter configuration in shared memory to later share with child processes */
    sconfig = (struct starterConfig *)mmap(NULL, sizeof(struct starterConfig), PROT_READ | PROT_WRITE, MAP_ANONYMOUS | MAP_SHARED, -1, 0);
    if ( sconfig == MAP_FAILED ) {
        fatalf("Memory allocation failed: %s\n", strerror(errno));
    }

    sconfig->starter.isSuid = is_suid();

    /* temporarily drop privileges while running as setuid */
    if ( sconfig->starter.isSuid ) {
        priv_drop(false);
    }

    debugf("Read engine configuration\n");

    /* read engine configuration from pipe */
    if ( ( sconfig->engine.size = read(pipe_fd, sconfig->engine.config, MAX_JSON_SIZE - 1) ) <= 0 ) {
        fatalf("Read engine configuration from pipe failed: %s\n", strerror(errno));
    }
    close(pipe_fd);

    /* fix I/O streams to point to /dev/null if they are closed */
    fix_streams();

    /* save opened file descriptors that won't be closed when stage 1 exits */
    master_fds = list_fd();

    /* set an invalid value for check */
    sconfig->starter.workingDirectoryFd = -1;

    /*
     *  CLONE_FILES will share file descriptors opened during stage 1,
     *  this is a lazy implementation to avoid passing file descriptors
     *  between wrapper and stage 1 over unix socket.
     *  Engines stage 1 must explicitly call method KeepFileDescriptor
     *  with a file descriptor in order to keep it open during cleanup
     *  step below.
     */
    process = fork_ns(CLONE_FILES);
    if ( process == 0 ) {
        /*
         *  stage1 is responsible for singularity configuration file parsing,
         *  handling user input, reading capabilities, and checking what
         *  namespaces are required
         */
        if ( sconfig->starter.isSuid ) {
            /* drop privileges permanently */
            priv_drop(true);
        }
        /* continue execution with Go runtime in main_linux.go */
        set_parent_death_signal(SIGKILL);
        verbosef("Spawn stage 1\n");
        goexecute = STAGE1;
        return;
    } else if ( process < 0 ) {
        fatalf("Failed to spawn stage 1\n");
    }

    debugf("Wait completion of stage1\n");
    wait_child("stage 1", process, false);

    /* change current working directory if requested by stage 1 */
    if ( sconfig->starter.workingDirectoryFd >= 0 ) {
        debugf("Applying stage 1 working directory\n");
        if ( fchdir(sconfig->starter.workingDirectoryFd) < 0 ) {
            fatalf("Failed to change current working directory: %s\n", strerror(errno));
        }
    }

    /* close all unattended and not registered file descriptors opened in stage 1 */
    cleanup_fd(master_fds, &sconfig->starter);
    /* free previously allocated resources during list_fd call */
    free(master_fds->fds);
    free(master_fds);

    /* block SIGCHLD signal handled later by stage 2/master */
    debugf("Set child signal mask\n");
    sigemptyset(&mask);
    sigaddset(&mask, SIGCHLD);
    if ( sigprocmask(SIG_SETMASK, &mask, NULL) == -1 ) {
        fatalf("Blocked signals error: %s\n", strerror(errno));
    }

    /* is container requested to run as an instance (or daemon) */
    if ( sconfig->container.isInstance ) {
        verbosef("Run as instance\n");
        process = fork();
        if ( process == 0 ) {
            /* this is the master process */
            if ( setsid() < 0 ) {
                fatalf("Can't set session leader: %s\n", strerror(errno));
            }
            umask(0);
        } else {
            sigset_t usrmask;
            static struct sigaction action;

            action.sa_sigaction = (void *)&do_exit;
            action.sa_flags = SA_SIGINFO|SA_RESTART;

            sigemptyset(&usrmask);
            sigaddset(&usrmask, SIGUSR1);

            if ( sigprocmask(SIG_SETMASK, &usrmask, NULL) == -1 ) {
                fatalf("Blocked signals error: %s\n", strerror(errno));
            }
            /* master process will send SIGUSR1 to detach successfully */
            if ( sigaction(SIGUSR1, &action, NULL) < 0 ) {
                fatalf("Failed to install signal handler for SIGUSR1\n");
            }
            if ( sigprocmask(SIG_UNBLOCK, &usrmask, NULL) == -1 ) {
                fatalf("Unblock signals error: %s\n", strerror(errno));
            }
            /* loop until master process exits with error */
            wait_child("instance", process, true);

            /* we should never return from the previous wait_child call */
            exit(1);
        }
    }

    /* master socket will be used by master process and stage 2 process both in C and Go context */
    debugf("Create socketpair for master communication channel\n");
    if ( socketpair(AF_UNIX, SOCK_STREAM|SOCK_CLOEXEC, 0, master_socket) < 0 ) {
        fatalf("Failed to create communication socket: %s\n", strerror(errno));
    }

    /* create RPC sockets only if the container is created */
    if ( !sconfig->container.namespace.joinOnly ) {
        debugf("Create RPC socketpair for communication between stage 2 and RPC server\n");
        if ( socketpair(AF_UNIX, SOCK_STREAM|SOCK_CLOEXEC, 0, rpc_socket) < 0 ) {
            fatalf("Failed to create communication socket: %s\n", strerror(errno));
        }
    }

    userns = user_namespace_init(&sconfig->container.namespace);
    switch ( userns ) {
    case NO_NAMESPACE:
        /* user namespace not enabled, continue with privileged workflow */
        priv_escalate(true);
        break;
    case ENTER_NAMESPACE:
        if ( sconfig->starter.isSuid && !sconfig->starter.hybridWorkflow ) {
            fatalf("Running setuid workflow with user namespace is not allowed\n");
        }
        break;
    case CREATE_NAMESPACE:
        if ( !sconfig->starter.hybridWorkflow ) {
            if ( sconfig->starter.isSuid ) {
                fatalf("Running setuid workflow with user namespace is not allowed\n");
            }
            /* master and container processes lives in the same user namespace */
            if ( create_namespace(CLONE_NEWUSER) < 0 ) {
                fatalf("Failed to create user namespace: %s\n", strerror(errno));
            }
        } else {
            /*
             * hybrid workflow, master process lives in host user namespace with the ability
             * to escalate privileges while container process lives in its own user namespace
             */
            clone_flags |= CLONE_NEWUSER;
        }
        break;
    }

    /* as we fork in any case, we set clone flag to create pid namespace during fork */
    pidns = pid_namespace_init(&sconfig->container.namespace);
    if ( pidns == CREATE_NAMESPACE ) {
        clone_flags |= CLONE_NEWPID;
    }

    process = fork_ns(clone_flags);
    if ( process == 0 ) {
        /* in the user namespace without any privileges */
        if ( userns == CREATE_NAMESPACE ) {
            /* wait parent write user namespace mappings */
            if ( wait_event(master_socket[1]) < 0 ) {
                fatalf("Error while waiting event for user namespace mappings\n");
            }
        }

        /* at this stage we are PID 1 if PID namespace requested */
        set_parent_death_signal(SIGKILL);

        close(master_socket[0]);

        /* initialize remaining namespaces */
        network_namespace_init(&sconfig->container.namespace);
        uts_namespace_init(&sconfig->container.namespace);
        ipc_namespace_init(&sconfig->container.namespace);
        cgroup_namespace_init(&sconfig->container.namespace);

        /*
         * depending of engines, the master process may require to propagate mount point
         * inside container (eg: FUSE mount), additionally mount done in container namespace
         * are propagated to master process mount namespace
         */
        if ( sconfig->starter.masterPropagateMount && userns != ENTER_NAMESPACE ) {
            shared_mount_namespace_init(&sconfig->container.namespace);
            /* tell master to continue execution and join mount namespace */
            send_event(master_socket[1]);
            /* wait until master joined the shared mount namespace */
            if ( wait_event(master_socket[1]) < 0 ) {
                fatalf("Error while waiting event for shared mount namespace\n");
            }
            mount_namespace_init(&sconfig->container.namespace, true);
        } else {
            send_event(master_socket[1]);
            mount_namespace_init(&sconfig->container.namespace, false);
        }

        if ( !sconfig->container.namespace.joinOnly ) {
            close(rpc_socket[0]);

            /*
             * use CLONE_FS here, because we want that pivot_root/chroot
             * occurring in RPC server process also affect stage 2 process
             * which is the final container process
             */
            process = fork_ns(CLONE_FS);
            if ( process == 0 ) {
                set_parent_death_signal(SIGKILL);
                verbosef("Spawn RPC server\n");
                /* continue execution with Go runtime in main_linux.go */
                goexecute = RPC_SERVER;
                return;
            } else if ( process > 0 ) {
                /* stage 2 doesn't use RPC connection at all */
                close(rpc_socket[1]);

                /* wait RPC server exits before running container process */
                wait_child("rpc server", process, false);

                if ( sconfig->starter.hybridWorkflow && sconfig->starter.isSuid ) {
                    /* make /proc/self readable by user to join instance without SUID workflow */
                    if ( prctl(PR_SET_DUMPABLE, 1) < 0 ) {
                        fatalf("Failed to set process dumpable: %s\n", strerror(errno));
                    }
                }
            } else {
                fatalf("Fork failed: %s\n", strerror(errno));
            }
        } else {
            verbosef("Spawn stage 2\n");
            verbosef("Don't execute RPC server, joining instance\n");
        }

        /* continue execution with Go runtime in main_linux.go */
        apply_container_privileges(&sconfig->container.privileges);
        goexecute = STAGE2;
        return;
    } else if ( process > 0 ) {
        int cwdfd;

        verbosef("Spawn master process\n");
        sconfig->container.pid = process;

        /* case where we joined a PID namespace but create a new mount namespace (eg: kubernetes POD) */
        if ( pidns == ENTER_NAMESPACE && is_namespace_create(&sconfig->container.namespace, CLONE_NEWNS) ) {
            if ( enter_namespace("/proc/self/ns/pid", CLONE_NEWPID) < 0 ) {
                fatalf("Failed to enter in pid namespace: %s\n", strerror(errno));
            }
        }

        close(master_socket[1]);

        /*
         * go to /proc/<pid> to open mount namespace and set user mappings with relative paths,
         * before that we open current working directory to restore it later, we don't use
         * workingDirectoryFd because this file descriptor may have been closed by cleanup_fd
         */
        cwdfd = open(".", O_RDONLY | O_DIRECTORY);
        if ( cwdfd < 0 ) {
            fatalf("Failed to open current working directory: %s\n", strerror(errno));
        }
        chdir_to_proc_pid(sconfig->container.pid);

        /* user namespace created, write user mappings */
        if ( userns == CREATE_NAMESPACE ) {
            /* set user namespace mappings */
            if ( sconfig->starter.hybridWorkflow ) {
                if ( sconfig->starter.isSuid ) {
                    /*
                     * hybrid workflow requires privileges for user mappings, we also preserve user
                     * filesytem UID here otherwise we would get a permission denied error during
                     * user mappings setup. User filesystem UID will be restored below by setresuid
                     * call
                     */
                    priv_escalate(false);
                    setup_userns_mappings(&sconfig->container.privileges);
                } else {
                    /* use newuidmap/newgidmap as fallback for hybrid workflow */
                    setup_userns_mappings_external(&sconfig->container);
                    /*
                     * without setuid, we could not join mount namespace below, so
                     * we need to join the fakeroot user namespace first
                     */
                    if ( enter_namespace("ns/user", CLONE_NEWUSER) < 0 ) {
                        fatalf("Failed to enter in fakeroot user namespace: %s\n", strerror(errno));
                    }
                }
            } else {
                setup_userns_mappings(&sconfig->container.privileges);
            }
            send_event(master_socket[0]);
        }

        /* wait child finish namespaces initialization */
        if ( wait_event(master_socket[0]) < 0 ) {
            /* child has exited before sending data */
            wait_child("stage 2", sconfig->container.pid, true);
        }

        /* engine requested to propagate mount to container */
        if ( sconfig->starter.masterPropagateMount && userns != ENTER_NAMESPACE ) {
            /* join child shared mount namespace with relative path */
            if ( enter_namespace("ns/mnt", CLONE_NEWNS) < 0 ) {
                fatalf("Failed to enter in shared mount namespace: %s\n", strerror(errno));
            }
            send_event(master_socket[0]);
        }

        /* staying in /proc/pid could lead to "no such process" error, go to previous working directory */
        if ( fchdir(cwdfd) < 0 ) {
            fatalf("Failed to restore current working directory: %s\n", strerror(errno));
        }
        close(cwdfd);

        if ( sconfig->container.namespace.joinOnly ) {
            /* joining container, don't execute Go runtime, just wait that container process exit */
            if ( sconfig->starter.isSuid ) {
                priv_drop(true);
            }
            debugf("Wait stage 2 child process\n");
            wait_child("stage 2", sconfig->container.pid, true);
        } else {
            close(rpc_socket[1]);

            /*
             * container creation, keep saved uid to allow further privileges
             * escalation from master process, because container network requires
             * privileges
             */
            if ( sconfig->starter.isSuid && setresuid(uid, uid, 0) < 0 ) {
                fatalf("Failed to drop privileges\n");
            }

            /* continue execution with Go runtime in main_linux.go */
            goexecute = MASTER;
            return;
        }
    }
    fatalf("Failed to create container namespaces\n");
}
