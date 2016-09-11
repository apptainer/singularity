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
#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <string.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/mount.h>
#include <sys/wait.h>
#include <unistd.h>
#include <stdlib.h>
#include <sched.h>
#include <linux/limits.h>


#include "util/file.h"
#include "util/util.h"
#include "lib/message.h"
#include "lib/config_parser.h"
#include "lib/privilege.h"

static int enabled = -1;

int singularity_ns_user_enabled(void) {
    singularity_message(DEBUG, "Checking user namespace enabled: %d\n", enabled);
    return(enabled);
}

// Check to make sure we are SUID or error
void check_for_suid(void) {
    if ( ( is_owner("/proc/self/exe", 0) != 0 ) || ( is_suid("/proc/self/exe") < 0 ) ) {
        singularity_message(ERROR, "User namespace not supported, and program not running privileged.\n");
        ABORT(255);
    }
}


int singularity_ns_user_unshare(void) {

    if ( ( is_suid("/proc/self/exe") == 0 ) && ( is_owner("/proc/self/exe", 0)  == 0) ) {
        singularity_message(VERBOSE, "Not virtualizing user namespace: running SUID root\n");
        return(0);
    }
    
    if ( getuid() == 0 ) {
        singularity_message(VERBOSE3, "Not virtualizing USER namespace: running as root\n");
        return(0);
    }

    singularity_config_rewind();
    if ( singularity_config_get_bool("allow user ns", 1) <= 0 ) {
        singularity_message(VERBOSE2, "Not virtualizing USER namespace by configuration\n");
        return(0);
    }

#ifdef NS_CLONE_NEWUSER
    singularity_message(DEBUG, "Attempting to virtualize the USER namespace\n");
    if ( unshare(CLONE_NEWUSER) != 0 ) {
        singularity_message(VERBOSE3, "Not virtualizing USER namespace: runtime support failed (%d:%s)\n", errno, strerror(errno));
        check_for_suid();
        return(0); // Don't fail when host support doesn't exist
    }
#else
    singularity_message(WARNING, "Skipping USER namespace creation, support not available on host\n");
    check_for_suid();
    return(0);
#endif

    uid_t uid = singularity_priv_getuid();
    gid_t gid = singularity_priv_getgid();

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
        singularity_message(DEBUG, "Setting GID map to: '0 %i 1'\n", gid);
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
        singularity_message(DEBUG, "Setting UID map to: '0 %i 1'\n", uid);
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

    singularity_priv_userns_ready();
    enabled = 0;

    return(0);
}


