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
    CAP_MKNOD,
    CAP_CHOWN,
    CAP_FOWNER,
    CAP_SYS_CHROOT,
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

    singularity_message(DEBUG, "Entering in a restricted capability set\n");

    for ( caps_index = 0; caps_index <= CAP_LAST_CAP; caps_index++ ) {
        keep_cap = -1;
        for ( keep_index = 0; capabilities[keep_index] != NO_CAP; keep_index++ ) {
            if ( caps_index == capabilities[keep_index] ) {
                keep_cap = caps_index;
                break;
            }
        }
        if ( keep_cap < 0 ) {
            if ( prctl(PR_CAPBSET_DROP, caps_index) < 0 ) {
                singularity_message(ERROR, "Failed to drop capabilities\n");
                ABORT(255);
            }
        }
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
