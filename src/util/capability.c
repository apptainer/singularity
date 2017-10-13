/* 
 * Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
 * 
 * This software is licensed under a 3-clause BSD license.  Please
 * consult LICENSE file distributed with the sources of this project regarding
 * your rights to use or distribute this software.
 * 
 */


#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <fcntl.h>
#include <sys/types.h>
#include <sys/prctl.h>
#include <pwd.h>
#include <errno.h>
#include <string.h>
#include <stdio.h>
#include <grp.h>
#include <limits.h>
#include <sched.h>
#include <linux/capability.h>
#include <sys/syscall.h>

#ifdef SINGULARITY_SECUREBITS
#  include <linux/securebits.h>
#else
#  include "util/securebits.h"
#endif /* SINGULARITY_SECUREBITS */

#include "config.h"

#include "util/file.h"
#include "util/util.h"
#include "util/registry.h"
#include "util/privilege.h"
#include "util/message.h"
#include "util/config_parser.h"

#define NO_CAP  CAP_LAST_CAP + 1

/* Support only 64 bits sets, since kernel 2.6.25 */
#ifdef _LINUX_CAPABILITY_VERSION_3
#  define LINUX_CAPABILITY_VERSION  _LINUX_CAPABILITY_VERSION_3
#elif defined(_LINUX_CAPABILITY_VERSION_2)
#  define LINUX_CAPABILITY_VERSION  _LINUX_CAPABILITY_VERSION_2
#else
#  error Linux 64 bits capability set not supported
#endif /* _LINUX_CAPABILITY_VERSION_3 */

/*
    if uid != 0 -> no capabilities
    if uid = 0 -> default capabilities
    if uid = 0 and keep_privs -> all capabilities
    if uid = 0 and no_privs -> no capabilities
    if uid = 0 and build stage 2 -> minimal capabilities
*/

static __u32 default_capabilities[] = {
    CAP_SETUID,
    CAP_SETGID,
    CAP_SETPCAP,
    CAP_SYS_ADMIN,
    CAP_NET_ADMIN,
    CAP_MKNOD,
    CAP_CHOWN,
    CAP_FOWNER,
    CAP_SYS_CHROOT,
    CAP_SYS_PTRACE,
    CAP_DAC_READ_SEARCH,
    CAP_DAC_OVERRIDE,
    NO_CAP
};

static __u32 minimal_capabilities[] = {
    CAP_SETUID,
    CAP_SETGID,
    CAP_CHOWN,
    CAP_FOWNER,
    CAP_SYS_CHROOT,
    CAP_DAC_READ_SEARCH,
    CAP_DAC_OVERRIDE,
    NO_CAP
};

static __u32 no_capabilities[] = {
    NO_CAP
};

int capget(cap_user_header_t hdrp, cap_user_data_t datap) {
    return syscall(__NR_capget, hdrp, datap);
}

int capset(cap_user_header_t hdrp, const cap_user_data_t datap) {
    return syscall(__NR_capset, hdrp, datap);
}

int singularity_capability_keep_privs(void) {
    if ( getuid() == 0 && singularity_registry_get("KEEP_PRIVS") != NULL ) {
        return(1);
    }
    return(0);
}

int singularity_capability_no_privs(void) {
    if ( getuid() == 0 && singularity_registry_get("NO_PRIVS") != NULL ) {
        return(1);
    }
    return(0);
}

void singularity_capability_set_securebits(void) {
    if ( prctl(PR_SET_SECUREBITS, SECBIT_KEEP_CAPS|
                                  SECBIT_KEEP_CAPS_LOCKED|
                                  SECBIT_NOROOT|
                                  SECBIT_NOROOT_LOCKED|
                                  SECBIT_NO_SETUID_FIXUP|
                                  SECBIT_NO_SETUID_FIXUP_LOCKED) < 0) {
        singularity_message(ERROR, "Failed to set securebits\n");
        ABORT(255);
    }
}

void singularity_capability_set(__u32 *capabilities) {
    int caps_index;
    int keep_index;
    int keep_cap;
    int last_cap;
    __u32 pcap;
    __u32 mask;
    struct __user_cap_header_struct header;
    struct __user_cap_data_struct data[2];

    singularity_message(DEBUG, "Entering in a restricted capability set\n");

    header.version = LINUX_CAPABILITY_VERSION;
    header.pid = getpid();

    if ( capget(&header, data) < 0 ) {
        singularity_message(ERROR, "Failed to get processus capabilities\n");
        ABORT(255);
    }

    // We can't rely on CAP_LAST_CAP if singularity is compiled in a container by
    // example, host is ubuntu with a recent kernel and container is a centos 6
    // container, so CAP_LAST_CAP could be less than CAP_LAST_CAP host and we could
    // forget to drop some capabilities. So we take the MSB of permitted set

    singularity_message(DEBUG, "Determining highest capability of the running process\n");

    pcap = data[1].permitted;
    mask = 1 << 31;

    if ( pcap > 0 ) {
        last_cap = 63;
    } else {
        last_cap = 31;
        pcap = data[0].permitted;
    }

    while ( !(pcap & mask) ) {
        last_cap -= 1;
        mask >>= 1;
    }

    /* fallback and read bounding set */
    if ( last_cap == 63 ) {
        for ( caps_index = last_cap; caps_index >= 0; caps_index-- ) {
            if ( prctl(PR_CAPBSET_READ, caps_index) <= 0 ) {
                last_cap--;
            } else {
                break;
            }
        }
    }

    singularity_message(DEBUG, "Dropping capabilities in bounding set\n");
    for ( caps_index = 0; caps_index <= last_cap; caps_index++ ) {
        keep_cap = -1;
        for ( keep_index = 0; capabilities[keep_index] != NO_CAP; keep_index++ ) {
            if ( caps_index == capabilities[keep_index] ) {
                keep_cap = caps_index;
                break;
            }
        }
        if ( keep_cap < 0 ) {
            if ( prctl(PR_CAPBSET_DROP, caps_index) < 0 ) {
                singularity_message(ERROR, "Failed to drop bounding capabilities set\n");
                ABORT(255);
            }
        }
    }

    // drop all in inheritable set to force childs to inherit capabilities from bounding set
    data[0].inheritable = 0;
    data[1].inheritable = 0;

    if ( capset(&header, data) < 0 ) {
        singularity_message(ERROR, "Failed to set processus capabilities\n");
        ABORT(255);
    }
}

void singularity_capability_init(void) {
    if ( ! singularity_capability_keep_privs() ) {
        singularity_capability_set(default_capabilities);
    }
}

void singularity_capability_init_minimal(void) {
    singularity_capability_set(minimal_capabilities);
}

void singularity_capability_drop_all(void) {
    if ( singularity_capability_no_privs() || ( ! singularity_capability_keep_privs() && getuid() != 0 ) ) {
        singularity_message(DEBUG, "Drop all capabilities\n");
        singularity_capability_set_securebits();
        singularity_capability_set(no_capabilities);
    }
}
