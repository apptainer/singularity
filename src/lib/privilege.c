/* 
 * Copyright (c) 2015-2016, Gregory M. Kurtzer. All rights reserved.
 * 
 * “Singularity” Copyright (c) 2016, The Regents of the University of California,
 * through Lawrence Berkeley National Laboratory (subject to receipt of any
 * required approvals from the U.S. Dept. of Energy).  All rights reserved.
 * 
 * This software is licensed under a customized 3-clause BSD license.  Please
 * consult LICENSE file distributed with the sources of this project regarding
 * your rights to use or distribute this software.
 * 
 * NOTICE.  This Software was developed under funding from the U.S. Department of
 * Energy and the U.S. Government consequently retains certain rights. As such,
 * the U.S. Government has been granted for itself and others acting on its
 * behalf a paid-up, nonexclusive, irrevocable, worldwide license in the Software
 * to reproduce, distribute copies to the public, prepare derivative works, and
 * perform publicly and display publicly, and to permit other to do so. 
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

#include "lib/privilege.h"
#include "config.h"
#include "util/file.h"
#include "util/util.h"
#include "lib/message.h"
//#include "singularity.h"



static struct PRIV_INFO {
    int ready;
    uid_t uid;
    gid_t gid;
    gid_t *gids;
    size_t gids_count;
    int userns_ready;
    int disable_setgroups;
    uid_t orig_uid;
    uid_t orig_gid;
    pid_t orig_pid;
    int target_mode;  // Set to 1 if we are running in "target mode" (admin specifies UID/GID)
} uinfo;


void singularity_priv_init(void) {
    long int target_uid = -1;
    long int target_gid = -1;
    memset(&uinfo, '\0', sizeof(uinfo));

    singularity_message(DEBUG, "Called singularity_priv_init(void)\n");

    if ( getuid() == 0 ) {
        char *target_uid_str = envar("SINGULARITY_TARGET_UID", "", 32); 
        char *target_gid_str = envar("SINGULARITY_TARGET_GID", "", 32); 
        if ( target_uid_str && !target_gid_str ) {
            singularity_message(ERROR, "A target UID is set (%s) but a target GID is not set (SINGULARITY_TARGET_GID).  Both must be specified.\n", target_uid_str);
            ABORT(255);
        }
        if (target_uid_str) {
            if ( -1 == str2int(target_uid_str, &target_uid) ) {
                singularity_message(ERROR, "Unable to convert target UID (%s) to integer: %s\n", target_uid_str, strerror(errno));
                ABORT(255);
            }
            if (target_uid < 500) {
                singularity_message(ERROR, "Target UID (%ld) must be 500 or greater to avoid system users.\n", target_uid);
                ABORT(255);
            }
            if (target_uid > UINT_MAX) { // Avoid anything greater than the traditional overflow UID.
                singularity_message(ERROR, "Target UID (%ld) cannot be greater than UINT_MAX.\n", target_uid);
                ABORT(255);
            }
        }
        if ( !target_uid_str && target_gid_str ) {
            singularity_message(ERROR, "A target GID is set (%s) but a target UID is not set (SINGULARITY_TARGET_UID).  Both must be specified.\n", target_gid_str);
            ABORT(255);
        }
        if (target_gid_str) {
            if ( -1 == str2int(target_gid_str, &target_gid) ) {
                singularity_message(ERROR, "Unable to convert target GID (%s) to integer: %s\n", target_gid_str, strerror(errno));
                ABORT(255);
            }
            if (target_gid < 500) {
                singularity_message(ERROR, "Target GID (%ld) must be 500 or greater to avoid system groups.\n", target_gid);
                ABORT(255);
            }
            if (target_gid > UINT_MAX) { // Avoid anything greater than the traditional overflow GID.
                singularity_message(ERROR, "Target GID (%ld) cannot be greater than UINT_MAX.\n", target_gid);
                ABORT(255);
            }
        }
        free(target_uid_str);
        free(target_gid_str);
    }
    if ( (target_uid >= 500) && (target_gid >= 500) ) {
        uinfo.target_mode = 1;
        uinfo.uid = target_uid;
        uinfo.gid = target_gid;
        uinfo.gids_count = 0;
        uinfo.gids = NULL;
    } else {

        uinfo.uid = getuid();
        uinfo.gid = getgid();
        uinfo.gids_count = getgroups(0, NULL);

        uinfo.gids = (gid_t *) malloc(sizeof(gid_t) * uinfo.gids_count);

        if ( getgroups(uinfo.gids_count, uinfo.gids) < 0 ) {
            singularity_message(ERROR, "Could not obtain current supplementary group list: %s\n", strerror(errno));
            ABORT(255);
        }
    }
    uinfo.ready = 1;

    singularity_message(DEBUG, "Returning singularity_priv_init(void)\n");
}

void singularity_priv_escalate(void) {

    if ( uinfo.ready != 1 ) {
        singularity_message(ERROR, "User info is not available\n");
        ABORT(255);
    }

    if ( uinfo.userns_ready == 1 ) {
        singularity_message(DEBUG, "Not escalating privileges, user namespace enabled\n");
        return;
    }

    if ( uinfo.uid == 0 ) {
        singularity_message(DEBUG, "Running as root, not changing privileges\n");
        return;
    }


    singularity_message(DEBUG, "Temporarily escalating privileges (U=%d)\n", getuid());

    if ( ( seteuid(0) < 0 ) || ( setegid(0) < 0 ) ) {
        singularity_message(ERROR, "The feature you are requesting requires privilege you do not have\n");
        ABORT(255);
    }

}

void singularity_priv_drop(void) {

    if ( uinfo.ready != 1 ) {
        singularity_message(ERROR, "User info is not available\n");
        ABORT(255);
    }

    if ( uinfo.userns_ready == 1 ) {
        singularity_message(DEBUG, "Not dropping privileges, user namespace enabled\n");
        return;
    }

    if ( uinfo.uid == 0 ) {
        singularity_message(DEBUG, "Running as root, not changing privileges\n");
        return;
    }


    singularity_message(DEBUG, "Dropping privileges to UID=%d, GID=%d\n", uinfo.uid, uinfo.gid);

    if ( setegid(uinfo.gid) < 0 ) {
        singularity_message(ERROR, "Could not drop effective group privileges to gid %d: %s\n", uinfo.gid, strerror(errno));
        ABORT(255);
    }

    if ( seteuid(uinfo.uid) < 0 ) {
        singularity_message(ERROR, "Could not drop effective user privileges to uid %d: %s\n", uinfo.uid, strerror(errno));
        ABORT(255);
    }

    singularity_message(DEBUG, "Confirming we have correct UID/GID\n");
    if ( getgid() != uinfo.gid ) {
        if ( uinfo.target_mode && getgid() != 0 ) {
            singularity_message(ERROR, "Non-zero real GID for target mode: %d\n", getgid());
                ABORT(255);
            } else if ( !uinfo.target_mode )
            {
                singularity_message(ERROR, "Failed to drop effective group privileges to gid %d (currently %d)\n", uinfo.gid, getgid());
                ABORT(255);
            }
        }

        if ( getuid() != uinfo.uid ) {
        if ( uinfo.target_mode && getuid() != 0 ) {
            singularity_message(ERROR, "Non-zero real UID for target mode: %d\n", getuid());
            ABORT(255);
        } else if ( !uinfo.target_mode ) {
            singularity_message(ERROR, "Failed to drop effective user privileges to uid %d (currently %d)\n", uinfo.uid, getuid());
            ABORT(255);
        }
    }

}

void singularity_priv_drop_perm(void) {
    singularity_message(DEBUG, "Called singularity_priv_drop_perm(void)\n");

    if ( uinfo.ready != 1 ) {
        singularity_message(ERROR, "User info is not available\n");
        ABORT(255);
    }

    if ( uinfo.userns_ready == 1 ) {
        singularity_message(VERBOSE2, "User namespace called, no privilges to drop\n");
        return;
    }

    if ( uinfo.uid == 0 ) {
        singularity_message(VERBOSE2, "Calling user is root, no privileges to drop\n");
        return;
    }

    singularity_message(DEBUG, "Escalating permissison so we can properly drop permission\n");
    singularity_priv_escalate();

    singularity_message(DEBUG, "Resetting supplementary groups\n");
    if ( setgroups(uinfo.gids_count, uinfo.gids) < 0 ) {
        singularity_message(ERROR, "Could not reset supplementary group list: %s\n", strerror(errno));
        ABORT(255);
    }

    singularity_message(DEBUG, "Dropping to group ID '%d'\n", uinfo.gid);
    if ( setgid(uinfo.gid) < 0 ) {
        singularity_message(ERROR, "Could not dump group privileges: %s\n", strerror(errno));
        ABORT(255);
    }

    singularity_message(DEBUG, "Dropping real and effective privileges to GID = '%d'\n", uinfo.gid);
    if ( setregid(uinfo.gid, uinfo.gid) < 0 ) {
        singularity_message(ERROR, "Could not dump real and effective group privileges: %s\n", strerror(errno));
        ABORT(255);
    }

    singularity_message(DEBUG, "Dropping real and effective privileges to UID = '%d'\n", uinfo.uid);
    if ( setreuid(uinfo.uid, uinfo.uid) < 0 ) {
        singularity_message(ERROR, "Could not dump real and effective user privileges: %s\n", strerror(errno));
        ABORT(255);
    }

    singularity_message(DEBUG, "Confirming we have correct GID\n");
    if ( getgid() != uinfo.gid ) {
        singularity_message(ERROR, "Failed to drop effective group privileges to gid %d: %s\n", uinfo.gid, strerror(errno));
        ABORT(255);
    }

    singularity_message(DEBUG, "Confirming we have correct UID\n");
    if ( getuid() != uinfo.uid ) {
        singularity_message(ERROR, "Failed to drop effective user privileges to uid %d: %s\n", uinfo.uid, strerror(errno));
        ABORT(255);
    }

#ifdef SINGULARITY_NO_NEW_PRIVS
    // Prevent the following processes to increase privileges
    singularity_message(DEBUG, "Setting NO_NEW_PRIVS to prevent future privilege escalations.\n");
    if ( prctl(PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0) != 0 ) {
        singularity_message(ERROR, "Could not set NO_NEW_PRIVS safeguard: %s\n", strerror(errno));
        ABORT(255);
    }
#else  // SINGULARITY_NO_NEW_PRIVS
    singularity_message(VERBOSE2, "Not enabling NO_NEW_PRIVS flag due to lack of compile-time support.\n");
#endif


    singularity_message(DEBUG, "Finished dropping privileges\n");
}


int singularity_priv_userns_enabled(void) {
    return uinfo.userns_ready;
}

void singularity_priv_userns_ready(void) {
    uinfo.userns_ready = 1;
}

uid_t singularity_priv_getuid(void) {
    if ( !uinfo.ready ) {
        singularity_message(ERROR, "Invoked before privilege info initialized!\n");
        ABORT(255);
    }
    return uinfo.uid;
}

gid_t singularity_priv_getgid(void) {
    if ( !uinfo.ready ) {
        singularity_message(ERROR, "Invoked before privilege info initialized!\n");
        ABORT(255);
    }
    return uinfo.gid;
}

const gid_t *singularity_priv_getgids(void) {
    if ( !uinfo.ready ) {
        singularity_message(ERROR, "Invoked before privilege info initialized!\n");
        ABORT(255);
    }
    return uinfo.gids;
}

int singularity_priv_getgidcount(void) {
    if ( !uinfo.ready ) {
        singularity_message(ERROR, "Invoked before privilege info initialized!\n");
        ABORT(255);
    }
    return uinfo.gids_count;
}

