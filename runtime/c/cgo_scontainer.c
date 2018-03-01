#define _GNU_SOURCE

#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/prctl.h>
#include <signal.h>
#include <string.h>
#include <errno.h>
#include <linux/securebits.h>
#include <linux/capability.h>
#include <sys/syscall.h>

#include "include/wrapper.h"

#include "util/message.c"

// Support only 64 bits sets, since kernel 2.6.25
#define CAPSET_MAX          40

#ifdef _LINUX_CAPABILITY_VERSION_3
#  define LINUX_CAPABILITY_VERSION  _LINUX_CAPABILITY_VERSION_3
#elif defined(_LINUX_CAPABILITY_VERSION_2)
#  define LINUX_CAPABILITY_VERSION  _LINUX_CAPABILITY_VERSION_2
#else
#  error Linux 64 bits capability set not supported
#endif // _LINUX_CAPABILITY_VERSION_3


static int capget(cap_user_header_t hdrp, cap_user_data_t datap) {
    return syscall(__NR_capget, hdrp, datap);
}

static int capset(cap_user_header_t hdrp, const cap_user_data_t datap) {
    return syscall(__NR_capset, hdrp, datap);
}

char *json_conf = NULL;
struct cConfig cconf;
pid_t child_stage2 = 0;

//
// drop privileges here to restrain users to access sensitive
// resources in /proc/<pid> during container setup
//
__attribute__((constructor)) static void init(void) {
    uid_t uid = getuid();
    gid_t gid = getgid();
    struct __user_cap_header_struct header;
    struct __user_cap_data_struct data[2];
    int stage = strtoul(getenv("SCONTAINER_STAGE"), NULL, 10);
    int fd = strtoul(getenv("SCONTAINER_SOCKET"), NULL, 10);
    int ret;

    if ( stage <= 0 ) {
        pfatal("STAGE environement variable not set");
    }
    if ( fd < 0 ) {
        pfatal("SOCKET environment variable not set");
    }

    print(DEBUG, "Entering in scontainer stage %d", stage);

    if ( prctl(PR_SET_PDEATHSIG, SIGKILL) < 0 ) {
        pfatal("Failed to set parent death signal");
    }

    print(DEBUG, "Read C runtime configuration for stage %d", stage);

    if ( (ret = read(fd, &cconf, sizeof(cconf))) != sizeof(cconf) ) {
        pfatal("read failed %d %d", ret, fd);
    }

    json_conf = (char *)malloc(cconf.jsonConfSize);
    if ( json_conf == NULL ) {
        pfatal("memory allocation failed");
    }

    print(DEBUG, "Read JSON runtime configuration for stage %d", stage);

    if ( (ret = read(fd, json_conf, cconf.jsonConfSize)) != cconf.jsonConfSize ) {
        pfatal("read json configuration failed");
    }

    if ( stage == 2 ) {
        child_stage2 = fork();
    }

    if ( child_stage2 < 0 ) {
        pfatal("Failed to spawn child");
    }

    if ( cconf.userNS == 1 || cconf.isSuid == 0 ) {
        return;
    }

    header.version = LINUX_CAPABILITY_VERSION;
    header.pid = 0;

    if ( capget(&header, data) < 0 ) {
        pfatal("Failed to get processus capabilities");
    }

    if ( child_stage2 > 0 ) {
        data[1].inheritable = (__u32)(cconf.capInheritable >> 32);
        data[0].inheritable = (__u32)(cconf.capInheritable & 0xFFFFFFFF);
        data[1].permitted = (__u32)(cconf.capPermitted >> 32);
        data[0].permitted = (__u32)(cconf.capPermitted & 0xFFFFFFFF);
        data[1].effective = (__u32)(cconf.capEffective >> 32);
        data[0].effective = (__u32)(cconf.capEffective & 0xFFFFFFFF);
    } else {
        data[1].inheritable = data[1].permitted = data[1].effective = 0;
        data[0].inheritable = data[0].permitted = data[0].effective = 0;
        cconf.capBounding = 0;
        cconf.capAmbient = 0;
    }

    if ( prctl(PR_SET_SECUREBITS, SECBIT_NO_SETUID_FIXUP|SECBIT_NO_SETUID_FIXUP_LOCKED) < 0 ) {
        pfatal("securebits: %s", strerror(errno));
    }

    if ( setresgid(gid, gid, gid) < 0 ) {
        pfatal("error gid");
    }
    if ( setresuid(uid, uid, uid) < 0 ) {
        pfatal("error uid");
    }

    if ( prctl(PR_SET_PDEATHSIG, SIGKILL) < 0 ) {
        pfatal("failed to set parent death signal");
    }

    int last_cap;
    for ( last_cap = CAPSET_MAX; ; last_cap-- ) {
        if ( prctl(PR_CAPBSET_READ, last_cap) > 0 || last_cap == 0 ) {
            break;
        }
    }

    int caps_index;
    for ( caps_index = 0; caps_index <= last_cap; caps_index++ ) {
        if ( !(cconf.capBounding & (1ULL << caps_index)) ) {
            if ( prctl(PR_CAPBSET_DROP, caps_index) < 0 ) {
                pfatal("Failed to drop bounding capabilities set: %s", strerror(errno));
            }
        }
    }

#ifdef PR_SET_NO_NEW_PRIVS
    if ( cconf.noNewPrivs ) {
        if ( prctl(PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0) < 0 ) {
            pfatal("Failed to set no new privs flag: %s", strerror(errno));
        }
    }
#endif

    if ( capset(&header, data) < 0 ) {
        pfatal("Failed to set processus capabilities");
    }

#ifdef PR_CAP_AMBIENT
    // set ambient capabilities if supported
    int i;
    for (i = 0; i <= CAPSET_MAX; i++ ) {
        if ( (cconf.capAmbient & (1ULL << i)) ) {
            if ( prctl(PR_CAP_AMBIENT, PR_CAP_AMBIENT_RAISE, i, 0, 0) < 0 ) {
                pfatal("Failed to set ambient capability: %s", strerror(errno));
            }
        }
    }
#endif
}
