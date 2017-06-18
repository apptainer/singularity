/* 
 * Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
 *
 * Copyright (c) 2015-2017, Gregory M. Kurtzer. All rights reserved.
 * 
 * Copyright (c) 2016-2017, The Regents of the University of California,
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

#include "config.h"

#include "util/file.h"
#include "util/util.h"
#include "util/registry.h"
#include "util/privilege.h"
#include "util/message.h"
#include "util/config_parser.h"


static struct PRIV_INFO {
    int ready;
    uid_t uid;
    gid_t gid;
    gid_t *gids;
    size_t gids_count;
    int userns_ready;
    uid_t orig_uid;
    uid_t orig_gid;
    pid_t orig_pid;
    char *home;
    char *homedir;
    char *username;
    int dropped_groups;
    int target_mode;  // Set to 1 if we are running in "target mode" (admin specifies UID/GID)
} uinfo;


// Cache of UID / GID of the 'singularity' user.
static struct SINGULARITY_PRIV_INFO {
    int ready;
    uid_t uid;
    gid_t gid;
} sinfo;




void singularity_priv_init(void) {
    long int target_uid = -1;
    long int target_gid = -1;
    memset(&uinfo, '\0', sizeof(uinfo));
    memset(&sinfo, '\0', sizeof(sinfo));
    char *home_tmp = singularity_registry_get("HOME");
    char *target_uid_str = singularity_registry_get("TARGET_UID");
    char *target_gid_str = singularity_registry_get("TARGET_GID");
    struct passwd *pwent;

    singularity_message(DEBUG, "Initializing user info\n");

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
    if ( (target_uid >= 500) && (target_gid >= 500) ) {
        if ( getuid() != 0 ) {
            singularity_message(ERROR, "Unable to use TARGET UID/GID mode when not running as root.\n");
            ABORT(255);
        }
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

    if ( ( pwent = getpwuid(uinfo.uid) ) == NULL ) {
        singularity_message(VERBOSE, "Failed obtaining user information for uid: %i\n", uinfo.uid);
        uinfo.username = strdup("NULL");
    } else {
        if ( ( uinfo.username = strdup(pwent->pw_name) ) != NULL ) {
            singularity_message(DEBUG, "Set the calling user's username to: %s\n", uinfo.username);
        } else {
            singularity_message(ERROR, "Failed obtaining the calling user's username\n");
            ABORT(255);
        }
    }

    singularity_message(DEBUG, "Marking uinfo structure as ready\n");
    uinfo.ready = 1;

    singularity_message(DEBUG, "Obtaining home directory\n");
    if ( home_tmp != NULL ) {
        char *colon = strchr(home_tmp, ':');

        if ( colon == NULL ) {
            uinfo.home = strdup(home_tmp);
            uinfo.homedir = uinfo.home;
            singularity_message(VERBOSE2, "Set home and homedir (via SINGULARITY_HOME) to: %s\n", uinfo.home);
        } else {
            *colon = '\0';
            uinfo.home = strdup(&colon[1]);
            singularity_message(VERBOSE2, "Set home (via SINGULARITY_HOME) to: %s\n", uinfo.home);
            uinfo.homedir = strdup(home_tmp);
            singularity_message(VERBOSE2, "Set the home directory (via SINGULARITY_HOME) to: %s\n", uinfo.homedir);
        }

    } else if ( pwent != NULL ) {
        if ( ( uinfo.home = strdup(pwent->pw_dir) ) != NULL ) {
            singularity_message(VERBOSE2, "Set home (via getpwuid()) to: %s\n", uinfo.home);
            uinfo.homedir = uinfo.home;
        } else {
            singularity_message(ERROR, "Failed obtaining the calling user's home directory\n");
            ABORT(255);
        }
    } else {
        uinfo.home = strdup("/");
        uinfo.homedir = uinfo.home;
    }
    
    return;
}


void singularity_priv_userns(void) {

    singularity_message(VERBOSE, "Invoking the user namespace\n");

    if ( singularity_config_get_bool(ALLOW_USER_NS) <= 0 ) {
        singularity_message(VERBOSE, "Not virtualizing USER namespace by configuration: 'allow user ns' = no\n");
    } else if ( getuid() == 0 ) {
        singularity_message(VERBOSE, "Not virtualizing USER namespace: running as root\n");
    } else if ( singularity_priv_is_suid() == 0 ) {
        singularity_message(VERBOSE, "Not virtualizing USER namespace: running as SUID\n");
    } else {
        uid_t uid = singularity_priv_getuid();
        gid_t gid = singularity_priv_getgid();

        singularity_message(DEBUG, "Attempting to virtualize the USER namespace\n");
        if ( unshare(CLONE_NEWUSER) != 0 ) {
            singularity_message(ERROR, "Failed invoking the NEWUSER namespace runtime: %s\n", strerror(errno));
            ABORT(255); // If we are configured to use CLONE_NEWUSER, we should abort if that fails
        }

        singularity_message(DEBUG, "Enabled user namespaces\n");

        {   
            singularity_message(DEBUG, "Setting setgroups to: 'deny'\n");
            char *map_file = (char *) malloc(PATH_MAX);
            snprintf(map_file, PATH_MAX-1, "/proc/%d/setgroups", getpid()); // Flawfinder: ignore
            FILE *map_fp = fopen(map_file, "w+"); // Flawfinder: ignore
            if ( map_fp != NULL ) {
                singularity_message(DEBUG, "Updating setgroups: %s\n", map_file);
                fprintf(map_fp, "deny\n");
                if ( fclose(map_fp) < 0 ) {
                    singularity_message(ERROR, "Failed to write deny to setgroup file %s: %s\n", map_file, strerror(errno));
                    ABORT(255);
                }
            } else {
                singularity_message(ERROR, "Could not write info to setgroups: %s\n", strerror(errno));
                ABORT(255);
            }
            free(map_file);
        }
        {
            singularity_message(DEBUG, "Setting GID map to: '%i %i 1'\n", gid, gid);
            char *map_file = (char *) malloc(PATH_MAX);
            snprintf(map_file, PATH_MAX-1, "/proc/%d/gid_map", getpid()); // Flawfinder: ignore
            FILE *map_fp = fopen(map_file, "w+"); // Flawfinder: ignore
            if ( map_fp != NULL ) {
                singularity_message(DEBUG, "Updating the parent gid_map: %s\n", map_file);
                fprintf(map_fp, "%i %i 1\n", gid, gid);
                if ( fclose(map_fp) < 0 ) {
                    singularity_message(ERROR, "Failed to write to GID map %s: %s\n", map_file, strerror(errno));
                    ABORT(255);
                }
            } else {
                singularity_message(ERROR, "Could not write parent info to gid_map: %s\n", strerror(errno));
                ABORT(255);
            }
            free(map_file);
        }
        {   
            singularity_message(DEBUG, "Setting UID map to: '%i %i 1'\n", uid, uid);
            char *map_file = (char *) malloc(PATH_MAX);
            snprintf(map_file, PATH_MAX-1, "/proc/%d/uid_map", getpid()); // Flawfinder: ignore
            FILE *map_fp = fopen(map_file, "w+"); // Flawfinder: ignore
            if ( map_fp != NULL ) {
                singularity_message(DEBUG, "Updating the parent uid_map: %s\n", map_file);
                fprintf(map_fp, "%i %i 1\n", uid, uid);
                if ( fclose(map_fp) < 0 ) {
                    singularity_message(ERROR, "Failed to write to UID map %s: %s\n", map_file, strerror(errno));
                    ABORT(255);
                }
            } else {
                singularity_message(ERROR, "Could not write parent info to uid_map: %s\n", strerror(errno));
                ABORT(255);
            }
            free(map_file);
        }

        uinfo.userns_ready = 1;
    }

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

    singularity_message(DEBUG, "Clearing supplementary GIDs.\n");
    if ( setgroups(0, NULL) == -1 ) {
        singularity_message(ERROR, "Unable to clear the supplementary group IDs: %s (errno=%d).\n", strerror(errno), errno);
        ABORT(255);
    }
    uinfo.dropped_groups = 1;

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

    // If we escalated privileges to user singularity (!=0), we need to set the EUID back to 0 first before
    // we can switch back to the invoking user.
    if ( (geteuid() != 0) && (seteuid(0) < 0) ) {
        singularity_message(VERBOSE, "Could not restore EUID to 0: %s (errno=%d).\n", strerror(errno), errno);
    }

    singularity_message(DEBUG, "Dropping privileges to UID=%d, GID=%d (%lu supplementary GIDs)\n", uinfo.uid, uinfo.gid, uinfo.gids_count);

    singularity_message(DEBUG, "Restoring supplementary groups\n");
    if ( uinfo.dropped_groups && (setgroups(uinfo.gids_count, uinfo.gids) < 0) ) {
        singularity_message(ERROR, "Could not reset supplementary group list: %s\n", strerror(errno));
        ABORT(255);
    }
    uinfo.dropped_groups = 0;

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
        } else if ( !uinfo.target_mode ) {
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
        singularity_message(ERROR, "Could not reset supplementary group list (perm): %s\n", strerror(errno));
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

/* Return 0 if program is SUID, -1 if not SUID */
int singularity_priv_is_suid(void) {
    if ( ( is_suid("/proc/self/exe") == 0 ) && ( is_owner("/proc/self/exe", 0)  == 0) ) {
        return(0);
    } else {
        return(-1);
    }
}

char *singularity_priv_home(void) {
    if ( !uinfo.ready ) {
        singularity_message(ERROR, "Invoked before privilege info initialized!\n");
        ABORT(255);
    }
    return(strdup(uinfo.home));
}

char *singularity_priv_homedir(void) {
    if ( !uinfo.ready ) {
        singularity_message(ERROR, "Invoked before privilege info initialized!\n");
        ABORT(255);
    }
    return(strdup(uinfo.homedir));
}

char *singularity_priv_getuser(void) {
    if ( !uinfo.ready ) {
        singularity_message(ERROR, "Invoked before privilege info initialized!\n");
        ABORT(255);
    }
    return uinfo.username;
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

int singularity_priv_has_gid(gid_t gid) {
    if ( !uinfo.ready ) {
        singularity_message(ERROR, "Invoked singularity_priv_has_gid before privilege info initialized!\n");
        ABORT(255);
    }
    int gid_idx;
    for (gid_idx=0; gid_idx<uinfo.gids_count; gid_idx++) {
        if (uinfo.gids[gid_idx] == gid) {
            return 1;
        }
    }
    return 0;
}
