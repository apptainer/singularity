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

/* C and JSON configuration */
struct starterConfig *sconfig;

/* Socket process communication */
int rpc_socket[2] = {-1, -1};
int master_socket[2] = {-1, -1};

unsigned char execute;

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
    verbosef("Get root privileges\n");
    if ( seteuid(0) < 0 ) {
        fatalf("Failed to set effective UID to 0\n");
    }
}

static void priv_drop(void) {
    uid_t uid = getuid();
    verbosef("Drop root privileges\n");
    if ( seteuid(uid) < 0 ) {
        fatalf("Failed to set effective UID to %d\n", uid);
    }
}

static void set_parent_death_signal(int signo) {
    debugf("Set parent death signal to %d\n", signo);
    if ( prctl(PR_SET_PDEATHSIG, signo) < 0 ) {
        fatalf("Failed to set parent death signal\n");
    }
}

static int apply_container_privileges(struct container *container, unsigned char isSuid) {
    uid_t uid = getuid();
    struct __user_cap_header_struct header;
    struct __user_cap_data_struct data[2];
    struct privileges *privileges = &container->privileges;

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

    /* apply target UID/GID for root user */
    if ( uid == 0 && !is_namespace_create(&container->namespace, CLONE_NEWUSER) ) {
        if ( privileges->numGID != 0 || privileges->targetUID != 0 ) {
            if ( prctl(PR_SET_SECUREBITS, SECBIT_NO_SETUID_FIXUP|SECBIT_NO_SETUID_FIXUP_LOCKED) < 0 ) {
                fatalf("Failed to set securebits: %s\n", strerror(errno));
            }
        }

        if ( privileges->numGID != 0 ) {
            debugf("Clear additional group IDs\n");

            if ( setgroups(0, NULL) < 0 ) {
                fatalf("Unable to clear additional group IDs: %s\n", strerror(errno));
            }
        }

        if ( privileges->numGID >= 2 ) {
            debugf("Set additional group IDs\n");

            if ( setgroups(privileges->numGID-1, &privileges->targetGID[1]) < 0 ) {
                fatalf("Failed to set additional groups: %s\n", strerror(errno));
            }
        }
        if ( privileges->numGID >= 1 ) {
            gid_t targetGID = privileges->targetGID[0];

            debugf("Set main group ID\n");

            if ( setresgid(targetGID, targetGID, targetGID) < 0 ) {
                fatalf("Failed to set GID %d: %s\n", targetGID, strerror(errno));
            }
        }
        if ( privileges->targetUID != 0 ) {
            uid_t targetUID = privileges->targetUID;

            debugf("Set user ID to %d\n", targetUID);

            if ( setresuid(targetUID, targetUID, targetUID) < 0 ) {
                fatalf("Failed to drop privileges: %s\n", strerror(errno));
            }
        }
    } else if ( isSuid ) {
        if ( prctl(PR_SET_SECUREBITS, SECBIT_NO_SETUID_FIXUP|SECBIT_NO_SETUID_FIXUP_LOCKED) < 0 ) {
            fatalf("Failed to set securebits: %s\n", strerror(errno));
        }

        if ( setresuid(uid, uid, uid) < 0 ) {
            fatalf("Failed to drop privileges: %s\n", strerror(errno));
        }
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
        if ( !support_nsflag(CLONE_NEWNET) ) {
            warningf("Skipping network namespace creation, not supported\n");
            return(0);
        }
        verbosef("Create network namespace\n");
        break;
    case CLONE_NEWIPC:
        if ( !support_nsflag(CLONE_NEWIPC) ) {
            warningf("Skipping ipc namespace creation, not supported\n");
            return(0);
        }
        verbosef("Create ipc namespace\n");
        break;
    case CLONE_NEWNS:
        if ( !support_nsflag(CLONE_NEWNS) ) {
            warningf("Skipping mount namespace creation, not supported\n");
            return(0);
        }
        verbosef("Create mount namespace\n");
        break;
    case CLONE_NEWUTS:
        if ( !support_nsflag(CLONE_NEWUTS) ) {
            warningf("Skipping uts namespace creation, not supported\n");
            return(0);
        }
        verbosef("Create uts namespace\n");
        break;
    case CLONE_NEWUSER:
        if ( !support_nsflag(CLONE_NEWUTS) ) {
            warningf("Skipping user namespace creation, not supported\n");
            return(0);
        }
        verbosef("Create user namespace\n");
        break;
    case CLONE_NEWCGROUP:
        if ( !support_nsflag(CLONE_NEWCGROUP) ) {
            warningf("Skipping cgroup namespace creation, not supported\n");
            return(0);
        }
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
    int ret = -1;
    int ns_fd;

    switch(nstype) {
    case CLONE_NEWPID:
        if ( !support_nsflag(CLONE_NEWPID) ) {
            errno = EINVAL;
            return(-1);
        }
        verbosef("Entering in pid namespace\n");
        break;
    case CLONE_NEWNET:
        if ( !support_nsflag(CLONE_NEWNET) ) {
            errno = EINVAL;
            return(-1);
        }
        verbosef("Entering in network namespace\n");
        break;
    case CLONE_NEWIPC:
        if ( !support_nsflag(CLONE_NEWIPC) ) {
            errno = EINVAL;
            return(-1);
        }
        verbosef("Entering in ipc namespace\n");
    case CLONE_NEWNS:
        if ( !support_nsflag(CLONE_NEWNS) ) {
            errno = EINVAL;
            return(-1);
        }
        verbosef("Entering in mount namespace\n");
    case CLONE_NEWUTS:
        if ( !support_nsflag(CLONE_NEWUTS) ) {
            errno = EINVAL;
            return(-1);
        }
        verbosef("Entering in uts namespace\n");
    case CLONE_NEWUSER:
        if ( !support_nsflag(CLONE_NEWUSER) ) {
            errno = EINVAL;
            return(-1);
        }
        verbosef("Entering in user namespace\n");
    case CLONE_NEWCGROUP:
        if ( !support_nsflag(CLONE_NEWCGROUP) ) {
            errno = EINVAL;
            return(-1);
        }
        verbosef("Entering in cgroup namespace\n");
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

static void setup_userns_mappings(struct container *container, pid_t pid, const char *setgroup) {
    FILE *map_fp;
    int i;
    struct idMapping *uidmap;
    struct idMapping *gidmap;
    char *path = (char *)malloc(PATH_MAX);

    debugf("Write %s to set group file\n", setgroup);
    memset(path, 0, PATH_MAX);
    if ( snprintf(path, PATH_MAX-1, "/proc/%d/setgroups", pid) < 0 ) {
        fatalf("Failed to write path /proc/%d/setgroups in buffer\n", pid);
    }

    map_fp = fopen(path, "w+"); // Flawfinder: ignore
    if ( map_fp != NULL ) {
        fprintf(map_fp, "%s\n", setgroup);
        if ( fclose(map_fp) < 0 ) {
            fatalf("Failed to write %s to setgroup file: %s\n", setgroup, strerror(errno));
        }
    } else {
        fatalf("Could not write info to setgroups: %s\n", strerror(errno));
    }

    debugf("Write to GID map\n");
    memset(path, 0, PATH_MAX);
    if ( snprintf(path, PATH_MAX-1, "/proc/%d/gid_map", pid) < 0 ) {
        fatalf("Failed to write path /proc/%d/gid_map in buffer\n", pid);
    }

    map_fp = fopen(path, "w+"); // Flawfinder: ignore
    if ( map_fp != NULL ) {
        fprintf(map_fp, "%s", container->privileges.gidMap);
        if ( fclose(map_fp) < 0 ) {
            fatalf("Failed to write to GID map: %s\n", strerror(errno));
        }
    } else {
        fatalf("Could not write parent info to gid_map: %s\n", strerror(errno));
    }

    debugf("Write to UID map\n");
    memset(path, 0, PATH_MAX);
    if ( snprintf(path, PATH_MAX-1, "/proc/%d/uid_map", pid) < 0 ) {
        fatalf("Failed to write path /proc/%d/uid_map in buffer\n", pid);
    }

    map_fp = fopen(path, "w+"); // Flawfinder: ignore
    if ( map_fp != NULL ) {
        fprintf(map_fp, "%s", container->privileges.uidMap);
        if ( fclose(map_fp) < 0 ) {
            fatalf("Failed to write to UID map: %s\n", strerror(errno));
        }
    } else {
        fatalf("Could not write parent info to uid_map: %s\n", strerror(errno));
    }

    free(path);
}

static void setup_userns_identity(struct container *container) {
    uid_t uidMap = container->privileges.targetUID;
    gid_t gidMap = container->privileges.targetGID[0];

    if ( setgroups(0, NULL) < 0 ) {
        fatalf("Unabled to clear additional group IDs: %s\n", strerror(errno));
    }
    if ( setresgid(gidMap, gidMap, gidMap) < 0 ) {
        fatalf("Failed to change namespace group identity: %s\n", strerror(errno));
    }
    if ( setresuid(uidMap, uidMap, uidMap) < 0 ) {
        fatalf("Failed to change namespace user identity: %s\n", strerror(errno));
    }
}

static int user_namespace_init(struct namespace *nsconfig, unsigned char allowSuid) {
    if ( is_namespace_enter(nsconfig->user) ) {
        if ( !allowSuid ) {
            fatalf("Running setuid workflow with user namespace is not allowed\n");
        }
        if ( enter_namespace(nsconfig->user, CLONE_NEWUSER) < 0 ) {
            fatalf("Failed to enter in user namespace: %s\n", strerror(errno));
        }
        return ENTER_NAMESPACE;
    } else if ( is_namespace_create(nsconfig, CLONE_NEWUSER) ) {
        if ( !allowSuid ) {
            fatalf("Running setuid workflow with user namespace is not allowed\n");
        }
        verbosef("Create user namespace\n");
        return CREATE_NAMESPACE;
    }
    return NO_NAMESPACE;
}

static int shared_mount_namespace_init(struct namespace *nsconfig) {
    if ( !is_namespace_enter(nsconfig->mount) ) {
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
    return NO_NAMESPACE;
}

static int pid_namespace_init(struct namespace *nsconfig) {
    if ( is_namespace_enter(nsconfig->pid) ) {
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
    if ( is_namespace_enter(nsconfig->network) ) {
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
    if ( is_namespace_enter(nsconfig->uts) ) {
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
    if ( is_namespace_enter(nsconfig->ipc) ) {
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
    if ( is_namespace_enter(nsconfig->cgroup) ) {
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

static int mount_namespace_init(struct namespace *nsconfig, unsigned char masterPropagateMount) {
    if ( is_namespace_enter(nsconfig->mount) ) {
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

static unsigned char is_suid(void) {
    ElfW(auxv_t) *auxv;
    unsigned char suid = 0;
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

static struct fdlist *list_fd(void) {
    int i = 0;
    int fd_proc;
    DIR *dir;
    struct dirent *dirent;
    struct fdlist *fl = (struct fdlist *)malloc(sizeof(struct fdlist));

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

static void cleanup_fd(struct fdlist *master, struct starter *starter) {
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

    while ( ( dirent = readdir(dir ) ) ) {
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
                    debugf("Can't set FD_CLOEXEC on file descriptor %d: %s", starter->fds[i], strerror(errno));
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

static void event_stop(int fd) {
    unsigned long long counter;

    if ( read(fd, &counter, sizeof(counter)) != sizeof(counter) ) {
        fatalf("Failed to receive sync signal: %s\n", strerror(errno));
    }
}

static void event_start(int fd) {
    unsigned long long counter = 1;

    if ( write(fd, &counter, sizeof(counter)) != sizeof(counter) ) {
        fatalf("Failed to synchronize with master: %s\n", strerror(errno));
    }
}

static void fix_fsuid(uid_t uid) {
    setfsuid(uid);

    if ( setfsuid(uid) != uid ) {
        fatalf("Failed to set filesystem uid to %d\n", uid);
    }
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
                fatalf("Error while fixing IO streams: %s", strerror(errno));
            }
        }
    }

    if ( null > 2 ) {
        close(null);
    }
}

static char *dupenv(const char *env) {
    char *var = getenv(env);

    if ( var != NULL ) {
        return strdup(var);
    } else {
        fatalf("%s environment variable isn't set\n", env);
    }

    return NULL;
}

static void exit_with_status(const char *name, int status) {
    if ( WIFEXITED(status) ) {
        verbosef("%s exited with status %d\n", name, WEXITSTATUS(status));
        exit(WEXITSTATUS(status));
    } else if ( WIFSIGNALED(status) ) {
        verbosef("%s interrupted by signal number %d\n", name, WTERMSIG(status));
        kill(getpid(), WTERMSIG(status));
    }
    fatalf("%s exited with unknown status\n", name);
}

void do_exit(int sig) {
    if ( sig == SIGUSR1 ) {
        exit(0);
    }
    exit(1);
}

__attribute__((constructor)) static void init(void) {
    uid_t uid = getuid();
    gid_t gid = getgid();
    sigset_t mask;
    pid_t stage_pid;
    char *loglevel;
    char *pipe_fd_env;
    int status;
    int forkfd = -1;
    int pipe_fd = -1;
    int fork_flags = 0;
    int join_chroot = 0;
    int sync_pipe[2];
    struct pollfd fds[2];
    struct fdlist *master_fds;

#ifndef SINGULARITY_NO_NEW_PRIVS
    fatalf("Host kernel is outdated and does not support PR_SET_NO_NEW_PRIVS!\n");
#endif

    loglevel = dupenv("SINGULARITY_MESSAGELEVEL");

    pipe_fd_env = getenv("PIPE_EXEC_FD");
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

    verbosef("Container runtime\n");

    /* initialize starter configuration and share it with child processes */
    sconfig = (struct starterConfig *)mmap(NULL, sizeof(struct starterConfig), PROT_READ | PROT_WRITE, MAP_ANONYMOUS | MAP_SHARED, -1, 0);
    if ( sconfig == MAP_FAILED ) {
        fatalf("Memory allocation failed: %s\n", strerror(errno));
    }

    sconfig->starter.isSuid = is_suid();

    if ( sconfig->starter.isSuid ) {
        priv_drop();
    }

    /* reset environment variables */
    clearenv();

    if ( loglevel != NULL ) {
        setenv("SINGULARITY_MESSAGELEVEL", loglevel, 1);
        free(loglevel);
    }

    /* read json configuration from stdin */
    debugf("Read json configuration from pipe\n");

    if ( ( sconfig->engine.size = read(pipe_fd, sconfig->engine.config, MAX_JSON_SIZE - 1) ) <= 0 ) {
        fatalf("Read JSON configuration from pipe failed: %s\n", strerror(errno));
    }
    close(pipe_fd);

    /* fix streams to point to /dev/null if they are closed */
    fix_streams();

    /* save opened file descriptors that won't be closed when stage 1 returns */
    master_fds = list_fd();

    /* set an invalid value for check */
    sconfig->starter.workingDirectoryFd = -1;

    /*
     *  use CLONE_FILES to share file descriptors opened during stage 1,
     *  this is a lazy implementation to avoid passing file descriptors
     *  between wrapper and stage 1 over unix socket.
     *  This is required so that all processes works with same files/directories
     *  to minimize race conditions
     */
    stage_pid = fork_ns(CLONE_FILES);
    if ( stage_pid == 0 ) {
        set_parent_death_signal(SIGKILL);
        debugf("Entering stage 1\n");

        /*
         *  stage1 is responsible for singularity configuration file parsing, handle user input,
         *  read capabilities, check what namespaces is required.
         */
        if ( sconfig->starter.isSuid ) {
            priv_escalate();
            /*
             * since all starter configuration fields are set to 0, the stage 1
             * process will have no privileges and no capabilities, additionally
             * this process lives in host namespaces.
             */
            apply_container_privileges(&sconfig->container, sconfig->starter.isSuid);
        }
        verbosef("Spawn stage 1\n");
        execute = STAGE1;
        return;
    } else if ( stage_pid < 0 ) {
        fatalf("Failed to spawn stage 1\n");
    }

    debugf("Wait completion of stage1\n");
    if ( wait(&status) != stage_pid ) {
        fatalf("Could not wait child process %d\n", stage_pid);
    }

    if ( WIFEXITED(status) && WEXITSTATUS(status) != 0 ) {
        verbosef("stage 1 exited with status %d\n", WEXITSTATUS(status));
        exit(WEXITSTATUS(status));
    } else if ( WIFSIGNALED(status) ) {
        verbosef("stage 1 interrupted by signal number %d\n", WTERMSIG(status));
        kill(getpid(), WTERMSIG(status));
    }

    /* is stage 1 want to change current working directory ? */
    if ( sconfig->starter.workingDirectoryFd >= 0 ) {
        debugf("Applying stage 1 working directory\n");
        if ( fchdir(sconfig->starter.workingDirectoryFd) < 0 ) {
            fatalf("Failed to change current working directory: %s\n", strerror(errno));
        }
    }

    /* close all unattended and not registered file descriptors opened in stage 1 */
    cleanup_fd(master_fds, &sconfig->starter);
    free(master_fds->fds);
    free(master_fds);

    /* block SIGCHLD signal handled later by stage 2/master */
    debugf("Set child signal mask\n");
    sigemptyset(&mask);
    sigaddset(&mask, SIGCHLD);
    if (sigprocmask(SIG_SETMASK, &mask, NULL) == -1) {
        fatalf("Blocked signals error: %s\n", strerror(errno));
    }

    if ( sconfig->container.isInstance ) {
        verbosef("Run as instance\n");
        int forked = fork();
        if ( forked == 0 ) {
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
            sigaddset(&usrmask, SIGUSR2);

            if (sigprocmask(SIG_SETMASK, &usrmask, NULL) == -1) {
                fatalf("Blocked signals error: %s\n", strerror(errno));
            }
            if (sigaction(SIGUSR2, &action, NULL) < 0) {
                fatalf("Failed to install signal handler for SIGUSR2\n");
            }
            if (sigaction(SIGUSR1, &action, NULL) < 0) {
                fatalf("Failed to install signal handler for SIGUSR1\n");
            }
            if (sigprocmask(SIG_UNBLOCK, &usrmask, NULL) == -1) {
                fatalf("Unblock signals error: %s\n", strerror(errno));
            }
            while ( waitpid(forked, &status, 0) <= 0 ) {
                continue;
            }
            exit_with_status("instance", status);
        }
    }

    debugf("Create socketpair for master communication channel\n");
    if ( socketpair(AF_UNIX, SOCK_STREAM|SOCK_CLOEXEC, 0, master_socket) < 0 ) {
        fatalf("Failed to create communication socket: %s\n", strerror(errno));
    }

    unsigned char allowSuid = !sconfig->starter.isSuid;
    int ret = user_namespace_init(&sconfig->container.namespace, allowSuid);
    if ( ret == NO_NAMESPACE ) {
        /* user namespace not enabled, continue with privileged workflow */
        priv_escalate();
    } else if ( ret == ENTER_NAMESPACE && !sconfig->starter.masterPropagateMount ) {
        /* special case for OCI engine requiring to change identity to match the joined instance (NEED FIX) */
        setup_userns_identity(&sconfig->container);
    } else if ( ret == CREATE_NAMESPACE && sconfig->starter.masterPropagateMount ) {
        /* user namespace enabled, master and container processes lives in the same user namespace */
        if ( create_namespace(CLONE_NEWUSER) < 0 ) {
            fatalf("Failed to create user namespace: %s\n", strerror(errno));
        }
        setup_userns_mappings(&sconfig->container, getpid(), "deny");
    } else if ( ret == CREATE_NAMESPACE ) {
        /*
         * hybrid approach, master process lives in host user namespace with the ability
         * to escalate privileges while container process lives in its own user namespace
         */
        priv_escalate();

        fork_flags |= CLONE_NEWUSER;

        forkfd = eventfd(0, 0);
        if ( forkfd < 0 ) {
            fatalf("Failed to create fork sync pipe between master and child: %s\n", strerror(errno));
        }
    }

    /*
     * depending of engines, the master process may require to propagate mount point
     * inside container (eg: FUSE mount), additionally mount done in container namespace
     * are propagated to master process mount namespace
     */
    if ( sconfig->starter.masterPropagateMount ) {
        shared_mount_namespace_init(&sconfig->container.namespace);
    }

    if ( !sconfig->container.namespace.joinOnly ) {
        debugf("Create RPC socketpair for communication between stage 2 and RPC server\n");
        if ( socketpair(AF_UNIX, SOCK_STREAM|SOCK_CLOEXEC, 0, rpc_socket) < 0 ) {
            fatalf("Failed to create communication socket: %s\n", strerror(errno));
        }
    }

    if ( sconfig->starter.isSuid ) {
        /* Use setfsuid to address issue about root_squash filesystems option */
        fix_fsuid(uid);

        /* force kernel to load overlay module to ease detection later */
        if ( mount("none", "/", "overlay", MS_SILENT, "") < 0 ) {
            if ( errno != EINVAL ) {
                debugf("Overlay seems not supported by kernel\n");
            } else {
                debugf("Overlay seems supported by kernel\n");
            }
        }
    }

    /* sync master and near child with an eventfd */
    if ( pipe(sync_pipe) < 0 ) {
        fatalf("Failed to create sync pipe: %s\n", strerror(errno));
    }

    /* as we fork in any case, we set clone flag to create pid namespace during fork */
    if ( pid_namespace_init(&sconfig->container.namespace) == CREATE_NAMESPACE ) {
        fork_flags |= CLONE_NEWPID;
    }

    stage_pid = fork_ns(fork_flags);

    if ( stage_pid == 0 ) {
        /* at this stage we are PID 1 if PID namespace requested */
        set_parent_death_signal(SIGKILL);

        if ( forkfd >= 0 ) {
            // wait parent write user namespace mappings
            event_stop(forkfd);
            close(forkfd);

            setup_userns_identity(&sconfig->container);
        }

        close(master_socket[0]);

        network_namespace_init(&sconfig->container.namespace);

        uts_namespace_init(&sconfig->container.namespace);

        ipc_namespace_init(&sconfig->container.namespace);

        cgroup_namespace_init(&sconfig->container.namespace);

        mount_namespace_init(&sconfig->container.namespace, sconfig->starter.masterPropagateMount);

        close(sync_pipe[0]);
        sync_pipe[0] = 0;
        if ( write(sync_pipe[1], &sync_pipe[0], sizeof(int)) < 0 ) {
            fatalf("Failed to send sync event: %s\n", strerror(errno));
        }
        close(sync_pipe[1]);

        execute = STAGE2;

        if ( !sconfig->container.namespace.joinOnly ) {
            close(rpc_socket[0]);

            /*
             * fork is a convenient way to apply capabilities and privileges drop
             * from single thread context before entering in stage 2
             */
            int process = fork_ns(CLONE_FS);

            if ( process == 0 ) {
                verbosef("Spawn RPC server\n");
                execute = RPC_SERVER;
            } else if ( process > 0 ) {
                int status;

                apply_container_privileges(&sconfig->container, sconfig->starter.isSuid);

                if ( wait(&status) != process ) {
                    fatalf("Error while waiting RPC server: %s\n", strerror(errno));
                }
                if ( rpc_socket[1] != -1 ) {
                    close(rpc_socket[1]);
                }
            } else {
                fatalf("Fork failed: %s\n", strerror(errno));
            }
        } else {
            verbosef("Spawn stage 2\n");
            verbosef("Don't execute RPC server, joining instance\n");
            apply_container_privileges(&sconfig->container, sconfig->starter.isSuid);
        }
        return;
    } else if ( stage_pid > 0 ) {
        if ( is_namespace_enter(sconfig->container.namespace.pid) && is_namespace_create(&sconfig->container.namespace, CLONE_NEWNS) ) {
            if ( enter_namespace("/proc/self/ns/pid", CLONE_NEWPID) < 0 ) {
                fatalf("Failed to enter in pid namespace: %s\n", strerror(errno));
            }
        }

        if ( forkfd >= 0 ) {
            setup_userns_mappings(&sconfig->container, stage_pid, "allow");

            event_start(forkfd);
            close(forkfd);
        }

        sconfig->container.pid = stage_pid;

        verbosef("Spawn master process\n");

        close(master_socket[1]);

        // wait child finish namespaces initialization
        close(sync_pipe[1]);
        sync_pipe[1] = -1;
        if ( read(sync_pipe[0], &sync_pipe[1], sizeof(int)) < 0 ) {
            fatalf("Failed to receive sync event: %s\n", strerror(errno));
        }
        close(sync_pipe[0]);

        // value not set, child has exited before sending data
        if ( sync_pipe[1] == -1 ) {
            waitpid(stage_pid, &status, 0);
            exit_with_status("stage 2", status);
        }

        if ( sconfig->container.namespace.joinOnly ) {
            if ( sconfig->starter.isSuid && setresuid(uid, uid, uid) < 0 ) {
                fatalf("Failed to drop privileges permanently\n");
            }
            debugf("Wait stage 2 child process\n");
            waitpid(stage_pid, &status, 0);
            exit_with_status("stage 2", status);
        } else {
            close(rpc_socket[1]);

            if ( sconfig->starter.isSuid && setresuid(uid, uid, 0) < 0 ) {
                fatalf("Failed to drop privileges\n");
            }
            execute = MASTER;
            return;
        }
    }
    fatalf("Failed to create container namespaces\n");
}
