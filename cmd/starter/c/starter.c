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
#include <sys/statfs.h>
#include <signal.h>
#include <sched.h>
#include <setjmp.h>
#include <sys/syscall.h>
#include <net/if.h>
#include <sys/eventfd.h>
#include <sys/sysmacros.h>
#include <linux/magic.h>

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

#define capflag(x)  (1ULL << x)

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

/*
 * fork_ns child stack living in BSS section, instead of
 * allocating dynamically the stack during each call, a stack
 * of 4096 bytes is set, on x86_64 around 80 bytes are used for
 * the stack, 4096 should let enough room for Go runtime which
 * should jump on its own stack
 */
static fork_stack_t child_stack = {0};

/* child function called by clone to return directly to sigsetjmp in fork_ns */
__attribute__((noinline)) static int clone_fn(void *arg) {
    siglongjmp(*(sigjmp_buf *)arg, 0);
}

__attribute__ ((returns_twice)) __attribute__((noinline)) static int fork_ns(unsigned int flags) {
    sigjmp_buf env;

    /*
     * sigsetjmp return 0 when called directly, and will return 1
     * after siglongjmp call in clone_fn. We always save signal mask.
     * This is hack to make clone() behave like fork() where child
     * continue execution from the calling point.
     */
    if ( sigsetjmp(env, 1) ) {
        /* child process will return here after siglongjmp call in clone_fn */
        return 0;
    }
    /* parent process */
    return clone(clone_fn, child_stack.ptr, (SIGCHLD|flags), env);
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

/* set_parent_death_signal sets the signal that the calling process will get when its parent dies */
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

static struct capabilities *get_process_capabilities() {
    struct __user_cap_header_struct header;
    struct __user_cap_data_struct data[2];
    struct capabilities *current = (struct capabilities *)malloc(sizeof(struct capabilities));

    if ( current == NULL ) {
        fatalf("Could not allocate memory");
    }

    header.version = LINUX_CAPABILITY_VERSION;
    header.pid = 0;

    if ( capget(&header, data) < 0 ) {
        fatalf("Failed to get processus capabilities\n");
    }

    current->permitted = ((unsigned long long)data[1].permitted << 32) | data[0].permitted;
    current->effective = ((unsigned long long)data[1].effective << 32) | data[0].effective;
    current->inheritable = ((unsigned long long)data[1].inheritable << 32) | data[0].inheritable;

    return current;
}

static int get_last_cap(void) {
    int last_cap;
    for ( last_cap = CAPSET_MIN; last_cap <= CAPSET_MAX; last_cap++ ) {
        if ( prctl(PR_CAPBSET_READ, last_cap) < 0 ) {
            /* an error means that capability is not valid, take the last valid */
            break;
        }
    }
    return --last_cap;
}

static void apply_privileges(struct privileges *privileges, struct capabilities *current) {
    uid_t currentUID = getuid();
    uid_t targetUID = currentUID;
    struct __user_cap_header_struct header;
    struct __user_cap_data_struct data[2];
    int last_cap = get_last_cap();
    int caps_index;

    /* adjust capabilities based on the lastest capability supported by the system */
    for ( caps_index = last_cap + 1; caps_index <= CAPSET_MAX; caps_index++ ) {
        privileges->capabilities.effective &= ~capflag(caps_index);
        privileges->capabilities.permitted &= ~capflag(caps_index);
        privileges->capabilities.bounding &= ~capflag(caps_index);
        privileges->capabilities.inheritable &= ~capflag(caps_index);
        privileges->capabilities.ambient &= ~capflag(caps_index);
    }

    debugf("Effective capabilities:   0x%016llx\n", privileges->capabilities.effective);
    debugf("Permitted capabilities:   0x%016llx\n", privileges->capabilities.permitted);
    debugf("Bounding capabilities:    0x%016llx\n", privileges->capabilities.bounding);
    debugf("Inheritable capabilities: 0x%016llx\n", privileges->capabilities.inheritable);
#ifdef USER_CAPABILITIES
    debugf("Ambient capabilities:     0x%016llx\n", privileges->capabilities.ambient);
#endif

    /* compare requested effective set with the current permitted set */
    if ( (privileges->capabilities.effective & current->permitted) != privileges->capabilities.effective ) {
        fatalf(
            "Requesting capability set 0x%016llx while permitted capability set is 0x%016llx\n",
            privileges->capabilities.effective,
            current->permitted
        );
    }

    data[1].inheritable = (__u32)(privileges->capabilities.inheritable >> 32);
    data[0].inheritable = (__u32)(privileges->capabilities.inheritable & 0xFFFFFFFF);
    data[1].permitted = (__u32)(privileges->capabilities.permitted >> 32);
    data[0].permitted = (__u32)(privileges->capabilities.permitted & 0xFFFFFFFF);
    data[1].effective = (__u32)(privileges->capabilities.effective >> 32);
    data[0].effective = (__u32)(privileges->capabilities.effective & 0xFFFFFFFF);

    for ( caps_index = 0; caps_index <= last_cap; caps_index++ ) {
        if ( !(privileges->capabilities.bounding & capflag(caps_index)) ) {
            if ( prctl(PR_CAPBSET_DROP, caps_index) < 0 ) {
                fatalf("Failed to drop cap %d bounding capabilities set: %s\n", caps_index, strerror(errno));
            }
        }
    }

    /*
     * prevent capabilities from being adjusted by kernel when changing uid/gid,
     * we need to keep capabilities to apply container capabilities during capset call
     * and to set ambient capabilities. We can't use capset before changing uid/gid
     * because CAP_SETUID/CAP_SETGID could be already dropped
     */
    if ( prctl(PR_SET_SECUREBITS, SECBIT_KEEP_CAPS) < 0 ) {
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

    if ( privileges->noNewPrivs ) {
        if ( prctl(PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0) < 0 ) {
            fatalf("Failed to set no new privs flag: %s\n", strerror(errno));
        }
        if ( prctl(PR_GET_NO_NEW_PRIVS, 0, 0 ,0, 0) != 1 ) {
            fatalf("Aborting, failed to set no new privs flag: %s\n", strerror(errno));
        }
    }

    header.version = LINUX_CAPABILITY_VERSION;
    header.pid = 0;

    if ( capset(&header, data) < 0 ) {
        fatalf("Failed to set process capabilities\n");
    }

#ifdef USER_CAPABILITIES
    // set ambient capabilities if supported
    for ( caps_index = 0; caps_index <= last_cap; caps_index++ ) {
        if ( (privileges->capabilities.ambient & capflag(caps_index)) ) {
            if ( prctl(PR_CAP_AMBIENT, PR_CAP_AMBIENT_RAISE, caps_index, 0, 0) < 0 ) {
                fatalf("Failed to set ambient capability: %s\n", strerror(errno));
            }
        }
    }
#endif
}

static void set_rpc_privileges(void) {
    struct privileges *priv = (struct privileges *)malloc(sizeof(struct privileges));
    struct capabilities *current = get_process_capabilities();

    if ( priv == NULL ) {
        fatalf("Could not allocate memory\n");
    }

    memset(priv, 0, sizeof(struct privileges));

    priv->capabilities.effective = capflag(CAP_SYS_ADMIN);
    /*
     * for some operations like container decryption, overlay mount,
     * chroot and creation of loop devices, the following capabilities
     * must be in the permitted set:
     * - CAP_MKNOD
     * - CAP_SYS_CHROOT
     * - CAP_SETGID
     * - CAP_SETUID
     * - CAP_FOWNER
     * - CAP_DAC_OVERRIDE
     * - CAP_DAC_READ_SEARCH
     * - CAP_CHOWN
     * - CAP_IPC_LOCK
     * - CAP_SYS_PTRACE
     */
    priv->capabilities.permitted = current->permitted;
    /* required by cryptsetup */
    priv->capabilities.bounding = capflag(CAP_SYS_ADMIN);
    priv->capabilities.bounding |= capflag(CAP_IPC_LOCK);
    priv->capabilities.bounding |= capflag(CAP_MKNOD);

    debugf("Set RPC privileges\n");
    apply_privileges(priv, current);
    set_parent_death_signal(SIGKILL);

    free(priv);
    free(current);
}

static void set_master_privileges(void) {
    struct privileges *priv = (struct privileges *)malloc(sizeof(struct privileges));
    struct capabilities *current = get_process_capabilities();

    if ( priv == NULL ) {
        fatalf("could not allocate memory\n");
    }

    memset(priv, 0, sizeof(struct privileges));

    priv->capabilities.effective |= capflag(CAP_SETGID);
    priv->capabilities.effective |= capflag(CAP_SETUID);

    priv->capabilities.permitted = current->permitted;
    priv->capabilities.bounding = current->permitted;
    priv->capabilities.inheritable = current->permitted;

    debugf("Set master privileges\n");
    apply_privileges(priv, current);

    free(priv);
    free(current);
}

#define MSG_SIZE 1024

static char *nserror(int err, int nstype) {
    char *msg = (char *)malloc(MSG_SIZE);
    char *path = (char *)malloc(MAX_PATH_SIZE);
    char *ns = NULL;
    char *name = NULL;

    if ( msg == NULL || path == NULL ) {
        fatalf("could not allocate memory\n");
    }

    memset(msg, 0, MSG_SIZE);
    memset(path, 0, MAX_PATH_SIZE);

    switch(nstype) {
    case CLONE_NEWNET:
        name = "network";
        ns = "net";
        break;
    case CLONE_NEWIPC:
        name = "ipc";
        ns = name;
        break;
    case CLONE_NEWPID:
        name = "pid";
        ns = name;
        break;
    case CLONE_NEWNS:
        name = "mount";
        ns = "mnt";
        break;
    case CLONE_NEWUTS:
        name = "uts";
        ns = name;
        break;
    case CLONE_NEWUSER:
        name = "user";
        ns = name;
        break;
    case CLONE_NEWCGROUP:
        name = "cgroup";
        ns = name;
        break;
    }
    if ( err == EINVAL ) {
        snprintf(path, MAX_PATH_SIZE-1, "/proc/self/ns/%s", ns);
        if ( access(path, 0) < 0 ) {
            snprintf(msg, MSG_SIZE-1, "%s namespace not supported by your system", name);
        } else {
            snprintf(msg, MSG_SIZE-1, "%s namespace disabled", name);
        }
    } else if ( err == EUSERS ) {
        snprintf(msg, MSG_SIZE-1, "limit on the nesting depth of %s namespaces was exceeded", name);
    } else if ( err == ENOSPC ) {
        snprintf(path, MAX_PATH_SIZE-1, "/proc/sys/user/max_%s_namespaces", ns);
        if ( access(path, 0) == 0 ) {
            snprintf(msg, MSG_SIZE-1, "maximum number of %s namespaces exceeded, check %s", name, path);
        } else {
            snprintf(msg, MSG_SIZE-1, "limit on the nesting depth of %s namespaces was exceeded", name);
        }
    } else if ( err == EPERM ) {
        if ( nstype != CLONE_NEWUSER ) {
            snprintf(msg, MSG_SIZE-1, "%s namespace requires privileges, check Singularity installation", name);
        } else {
            snprintf(path, MAX_PATH_SIZE-1, "/proc/sys/kernel/unprivileged_userns_clone");
            if ( access(path, 0) == 0 ) {
                snprintf(msg, MSG_SIZE-1, "user namespace requires to set %s to 1", path);
            } else {
                snprintf(msg, MSG_SIZE-1, "not allowed to create user namespace");
            }
        }
    } else {
        free(msg);
        msg = strerror(err);
    }
    free(path);
    return msg;
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

    if ( xsetns(ns_fd, nstype) < 0 ) {
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
            fatalf("Failed to create network namespace: %s\n", nserror(errno, CLONE_NEWNET));
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
            fatalf("Failed to create uts namespace: %s\n", nserror(errno, CLONE_NEWUTS));
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
            fatalf("Failed to create ipc namespace: %s\n", nserror(errno, CLONE_NEWIPC));
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
            fatalf("Failed to create cgroup namespace: %s\n", nserror(errno, CLONE_NEWCGROUP));
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
                fatalf("Failed to create mount namespace: %s\n", nserror(errno, CLONE_NEWNS));
            }
            if ( propagation && mount(NULL, "/", NULL, propagation, NULL) < 0 ) {
                fatalf("Failed to set mount propagation: %s\n", strerror(errno));
            }
        } else {
            /* create a namespace for container process to separate master during pivot_root */
            if ( create_namespace(CLONE_NEWNS) < 0 ) {
                fatalf("Failed to create mount namespace: %s\n", nserror(errno, CLONE_NEWNS));
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
        fatalf("Failed to create mount namespace: %s\n", nserror(errno, CLONE_NEWNS));
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

/*
 * is_suid returns true if this binary has suid bit set or if it
 * has additional capabilities in extended file attributes
 */
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

/* list_fd returns list of currently opened file descriptors (from /proc/self/fd) */
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

/*
 * cleanup_fd closes all file descriptors that are not in
 * master's fdlist and not in starter's fds list as well.
 */
static void cleanup_fd(fdlist_t *master, struct starter *starter) {
    int fd_proc;
    DIR *dir;
    struct dirent *dirent;
    int i, fd;
    bool found;

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

        found = false;

        /* check if the file descriptor was open before stage 1 execution */
        for ( i = 0; i < master->num; i++ ) {
            if ( master->fds[i] == fd ) {
                found = true;
                break;
            }
        }
        if ( found ) {
            continue;
        }

        found = false;

        /* check if the file descriptor need to remain opened */
        for ( i = 0; i < starter->numfds; i++ ) {
            if ( starter->fds[i] == fd ) {
                found = true;
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

/* fix_streams makes closed stdin/stdout/stderr file descriptors point to /dev/null */
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

/* fix_userns_devfuse_fd reopens /dev/fuse file descriptors in a user namespace */
static void fix_userns_devfuse_fd(struct starter *starter) {
    struct stat st;
    int i, newfd, oldfd;

    for ( i = 0; i < starter->numfds; i++ ) {
        oldfd = starter->fds[i];
        if ( fstat(oldfd, &st) < 0 ) {
            fatalf("Failed to get file information for file descriptor %d: %s\n", oldfd, strerror(errno));
        }
        if ( major(st.st_rdev) == 10 && minor(st.st_rdev) == 229 ) {
            newfd = open("/dev/fuse", O_RDWR);
            if ( newfd < 0 ) {
                fatalf("Failed to open /dev/fuse: %s\n", strerror(errno));
            }
            if ( dup3(newfd, oldfd, O_CLOEXEC) < 0 ) {
                fatalf("Failed to duplicate file descriptor: %s\n", strerror(errno));
            }
            close(newfd);
        }
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

/* "noop" mount operation to force kernel to load overlay module */
void load_overlay_module(void) {
    if ( geteuid() == 0 && getenv("LOAD_OVERLAY_MODULE") != NULL ) {
        debugf("Trying to load overlay kernel module\n");
        if ( mount(NULL, "/", "overlay", MS_SILENT, NULL) < 0 ) {
            if ( errno == EINVAL ) {
                debugf("Overlay seems supported by the kernel\n");
            } else {
                debugf("Overlay seems not supported by the kernel\n");
            }
        }
    }
}

/* read engine configuration from environment variables */
static void read_engine_config(struct engine *engine) {
    char *engine_config[MAX_ENGINE_CONFIG_CHUNK];
    char env_key[ENGINE_CONFIG_ENV_PADDING] = {0};
    char *engine_chunk;
    unsigned long int nchunk = 0;
    off_t offset = 0;
    size_t length;
    int i;

    debugf("Read engine configuration\n");

    engine_chunk = getenv(ENGINE_CONFIG_CHUNK_ENV);
    if ( engine_chunk == NULL ) {
        fatalf("No engine config chunk provided\n");
    }
    nchunk = strtoul(engine_chunk, NULL, 10);
    if ( nchunk == 0 || nchunk > MAX_ENGINE_CONFIG_CHUNK ) {
        fatalf("Bad number of engine config chunk provided '%s': 0 or > %d\n", engine_chunk, MAX_ENGINE_CONFIG_CHUNK);
    }

    for ( i = 0; i < nchunk; i++ ) {
        snprintf(env_key, sizeof(env_key)-1, ENGINE_CONFIG_ENV"%d", i+1);
        engine_config[i] = getenv(env_key);
        if ( engine_config[i] == NULL ) {
            fatalf("No engine configuration found in %s\n", env_key);
        }
        engine->size += strnlen(engine_config[i], MAX_CHUNK_SIZE);
    }

    if ( engine->size >= MAX_ENGINE_CONFIG_SIZE ) {
        fatalf("Engine configuration too big >= %d bytes\n", MAX_ENGINE_CONFIG_SIZE);
    }

    /* allocate additional space for stage1 */
    engine->map_size = engine->size + MAX_CHUNK_SIZE;
    engine->config = (char *)mmap(NULL, engine->map_size, PROT_READ | PROT_WRITE, MAP_ANONYMOUS | MAP_SHARED, -1, 0);
    if ( engine->config == MAP_FAILED ) {
        fatalf("Memory allocation failed: %s\n", strerror(errno));
    }

    for ( i = 0; i < nchunk; i++ ) {
        length = strnlen(engine_config[i], MAX_CHUNK_SIZE);
        memcpy(&engine->config[offset], engine_config[i], length);
        offset += length;
    }
}

/* release previously mmap'ed memory */
static void release_memory(struct starterConfig *sconfig) {
    if ( munmap(sconfig->engine.config, sconfig->engine.map_size) < 0 ) {
        fatalf("Engine configuration memory release failed: %s\n", strerror(errno));
    }
    if ( munmap(sconfig, sizeof(struct starterConfig)) < 0 ) {
        fatalf("Starter configuration memory release failed: %s\n", strerror(errno));
    }
}

/*
 * Starter's entrypoint executed before Go runtime in a single-thread context.
 *
 * The constructor attribute causes init(void) function to be called automatically before
 * execution enters main(). This behavior is required in order to prepare isolated environment
 * for a container. Init will create and(or) enter requested namespaces delegating setup work
 * to the specific engine. Init forks oneself a couple of times during execution, which allows
 * engine to perform initialization inside the container context (RPC server) and outside of it
 * (CreateContainer method of an engine). At the end only two processes will be left: a container
 * process in the prepared environment and a master process which monitors container's state outside of it.
 */
__attribute__((constructor)) static void init(void) {
    uid_t uid = getuid();
    sigset_t mask;
    pid_t process;
    int clone_flags = 0;
    int userns = NO_NAMESPACE, pidns = NO_NAMESPACE;
    fdlist_t *master_fds;

    verbosef("Starter initialization\n");

#ifndef SINGULARITY_NO_NEW_PRIVS
    fatalf("Host kernel is outdated and does not support PR_SET_NO_NEW_PRIVS!\n");
#endif

    /* force loading overlay kernel module if requested */
    load_overlay_module();

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

    /* retrieve engine configuration from environment variables */
    read_engine_config(&sconfig->engine);

    /* cleanup environment variables */
    cleanenv();

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
        set_parent_death_signal(SIGKILL);
        verbosef("Spawn stage 1\n");
        goexecute = STAGE1;
        /* continue execution with Go runtime in main_linux.go */
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
            /*
             * this is the master process, also a daemon
             * detach it from the current session
             * the parent will exit so init will become master's parent
             */
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
        /*
         * user namespace not enabled, continue with privileged workflow
         * this will fail if starter is run without suid
         */
        if ( sconfig->starter.isSuid ) {
            priv_escalate(true);
        } else if ( uid != 0 ) {
            fatalf("No setuid installation found, for unprivileged installation use: ./mconfig --without-suid\n");
        }
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
                fatalf("Failed to create user namespace: %s\n", nserror(errno, CLONE_NEWUSER));
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
        struct capabilities *current = NULL;

        /* close master end of the communication socket */
        close(master_socket[0]);

        /* in the user namespace without any privileges */
        if ( userns == CREATE_NAMESPACE ) {
            /* re-open /dev/fuse file descriptors if any in the new user namespace */
            fix_userns_devfuse_fd(&sconfig->starter);

            /* wait parent write user namespace mappings */
            if ( wait_event(master_socket[1]) < 0 ) {
                fatalf("Error while waiting event for user namespace mappings: %s\n", strerror(errno));
            }
        }

        /* at this stage we are PID 1 if PID namespace requested */
        set_parent_death_signal(SIGKILL);

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
            /* close master end of rpc communication socket */
            close(rpc_socket[0]);

            /*
             * use CLONE_FS here, because we want that pivot_root/chroot
             * occurring in RPC server process also affect stage 2 process
             * which is the final container process
             */
            process = fork_ns(CLONE_FS);
            if ( process == 0 ) {
                if ( sconfig->starter.isSuid && geteuid() == 0 ) {
                    set_rpc_privileges();
                }
                verbosef("Spawn RPC server\n");
                goexecute = RPC_SERVER;
                /* continue execution with Go runtime in main_linux.go */
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

        debugf("Set container privileges\n");
        current = get_process_capabilities();
        apply_privileges(&sconfig->container.privileges, current);
        set_parent_death_signal(SIGKILL);
        free(current);

        goexecute = STAGE2;
        /* continue execution with Go runtime in main_linux.go */
        return;
    } else if ( process > 0 ) {
        int cwdfd;

        verbosef("Spawn master process\n");
        sconfig->container.pid = process;

        /* close container end of the communication socket */
        close(master_socket[1]);

        /*
         * case where we joined a PID namespace already,
         * but a new mount namespace was requested (e.g. kubernetes POD).
         * go back to the host's PID namespace in this case.
         */
        if ( pidns == ENTER_NAMESPACE && is_namespace_create(&sconfig->container.namespace, CLONE_NEWNS) ) {
            if ( enter_namespace("/proc/self/ns/pid", CLONE_NEWPID) < 0 ) {
                fatalf("Failed to enter in pid namespace: %s\n", strerror(errno));
            }
        }

        /*
         * go to /proc/<pid> to open mount namespace and set user mappings with relative paths,
         * before that we open current working directory to restore it later, we don't use
         * workingDirectoryFd because this file descriptor may have been closed by cleanup_fd
         */
        cwdfd = open(".", O_RDONLY | O_DIRECTORY);
        if ( cwdfd < 0 ) {
            fatalf("Failed to open current working directory: %s\n", strerror(errno));
        }

        /* user namespace created, write user mappings */
        if ( userns == CREATE_NAMESPACE ) {
            /* set user namespace mappings */
            if ( sconfig->starter.hybridWorkflow ) {
                if ( sconfig->starter.isSuid ) {
                    /*
                     * hybrid workflow requires privileges for user mappings, we also preserve user
                     * filesystem UID here otherwise we would get a permission denied error during
                     * user mappings setup. User filesystem UID will be restored below by setresuid
                     * call.
                     */
                    priv_escalate(false);
                    chdir_to_proc_pid(sconfig->container.pid);
                    setup_userns_mappings(&sconfig->container.privileges);
                } else {
                    chdir_to_proc_pid(sconfig->container.pid);
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
                chdir_to_proc_pid(sconfig->container.pid);
                setup_userns_mappings(&sconfig->container.privileges);
            }
            send_event(master_socket[0]);
        } else {
            chdir_to_proc_pid(sconfig->container.pid);
        }

        /* wait child finish namespaces initialization */
        if ( wait_event(master_socket[0]) < 0 ) {
            /* child has exited before sending data */
            wait_child("stage 2", sconfig->container.pid, true);
        }

        /* engine requested to propagate mount to container */
        if ( sconfig->starter.masterPropagateMount && userns != ENTER_NAMESPACE ) {
            struct stat rootfs, newrootfs;

            /* keep stat information for root filesystem comparison */
            if ( stat("/", &rootfs) < 0 ) {
                fatalf("Failed to get root directory information: %s", strerror(errno));
            }

            /* join child shared mount namespace with relative path */
            if ( enter_namespace("ns/mnt", CLONE_NEWNS) < 0 ) {
                fatalf("Failed to enter in shared mount namespace: %s\n", strerror(errno));
            }

            /* take stat information after namespace join */
            if ( stat("/", &newrootfs) < 0 ) {
                fatalf("Failed to get root directory information: %s", strerror(errno));
            }

            /*
             * we compare st_dev and st_ino to check if we are in the current root
             * filesystem, on some systems the mount namespace join above could escape the
             * current root filesystem when an init process (initrd, container or chrooted process)
             * do a chroot instead of switch_root for the initrd case or didn't use
             * pivot_root/mount MS_MOVE for the container solution case
             */
            if ( rootfs.st_dev != newrootfs.st_dev || rootfs.st_ino != newrootfs.st_ino ) {
                struct statfs fs;

                debugf("Root filesystem change detected, retrieving new root filesystem information\n");
                if ( statfs("/", &fs) < 0 ) {
                    fatalf("Failed to retrieve root filesystem information: %s", strerror(errno));
                }

                /* check if we are in the ram disk filesystem */
                if ( newrootfs.st_ino == 2 && (fs.f_type == RAMFS_MAGIC || fs.f_type == TMPFS_MAGIC) ) {
                    warningf("Initrd uses chroot instead of switch_root to setup the root filesystem\n");
                } else {
                    warningf("Running inside a weak chrooted environment, prefer pivot_root instead of chroot\n");
                }
                fatalf("Aborting as Singularity cannot run correctly without modifications to your environment\n");
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
            release_memory(sconfig);
            wait_child("stage 2", process, true);
        } else {
            /* close container end of rpc communication socket */
            close(rpc_socket[1]);

            /*
             * container creation, just keep setuid/setgid capabilities for
             * further privileges escalation and set UID/GID to the original
             * user when using setuid workflow
             */
            if ( sconfig->starter.isSuid && geteuid() == 0 ) {
                set_master_privileges();
            }

            goexecute = MASTER;
            /* continue execution with Go runtime in main_linux.go */
            return;
        }
    }
    if ( clone_flags & CLONE_NEWPID != 0 && clone_flags & CLONE_NEWUSER == 0 ) {
        fatalf("Failed to create container namespace: %s\n", nserror(errno, CLONE_NEWPID));
    } else if ( clone_flags & CLONE_NEWUSER != 0 ) {
        fatalf("Failed to create container namespace: %s\n", nserror(errno, CLONE_NEWUSER));
    }
    fatalf("Failed to create container process: %s\n", strerror(errno));
}
