/* 
 * Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
 * 
 * This software is licensed under a 3-clause BSD license.  Please
 * consult LICENSE file distributed with the sources of this project regarding
 * your rights to use or distribute this software.
 * 
 */

#define _GNU_SOURCE
#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <string.h>
#include <limits.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/mount.h>
#include <sys/wait.h>
#include <sys/ioctl.h>
#include <net/if.h>
#include <unistd.h>
#include <stdlib.h>
#include <sched.h>

#include "util/file.h"
#include "util/util.h"
#include "util/message.h"
#include "util/config_parser.h"
#include "util/privilege.h"
#include "util/fork.h"
#include "util/registry.h"
#include "util/daemon.h"
#include "util/setns.h"


int _singularity_runtime_ns_user(void) {

    if ( singularity_priv_userns_enabled() ) {
        uid_t uid = singularity_priv_getuid();
        gid_t gid = singularity_priv_getgid();
        char *target_uid_str = singularity_registry_get("USERNS_UID");
        char *target_gid_str = singularity_registry_get("USERNS_GID");
        long int target_uid = uid, target_gid = gid;

        singularity_message(VERBOSE, "Invoking the user namespace\n");

        if ( target_uid_str != NULL ) {
            if ( str2int(target_uid_str, &target_uid) < 0 ) {
                singularity_message(ERROR, "Unable to convert target UID (%s) to integer: %s\n", target_uid_str, strerror(errno));
                ABORT(255);
            }
        }
        if ( target_gid_str != NULL ) {
            if ( str2int(target_gid_str, &target_gid) < 0 ) {
                singularity_message(ERROR, "Unable to convert target GID (%s) to integer: %s\n", target_gid_str, strerror(errno));
                ABORT(255);
            }
        }

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
            singularity_message(DEBUG, "Setting GID map to: '%i %i 1'\n", (gid_t)target_gid, gid);
            char *map_file = (char *) malloc(PATH_MAX);
            snprintf(map_file, PATH_MAX-1, "/proc/%d/gid_map", getpid()); // Flawfinder: ignore
            FILE *map_fp = fopen(map_file, "w+"); // Flawfinder: ignore
            if ( map_fp != NULL ) {
                singularity_message(DEBUG, "Updating the parent gid_map: %s\n", map_file);
                fprintf(map_fp, "%i %i 1\n", (gid_t)target_gid, gid);
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
            singularity_message(DEBUG, "Setting UID map to: '%i %i 1'\n", (uid_t)target_uid, uid);
            char *map_file = (char *) malloc(PATH_MAX);
            snprintf(map_file, PATH_MAX-1, "/proc/%d/uid_map", getpid()); // Flawfinder: ignore
            FILE *map_fp = fopen(map_file, "w+"); // Flawfinder: ignore
            if ( map_fp != NULL ) {
                singularity_message(DEBUG, "Updating the parent uid_map: %s\n", map_file);
                fprintf(map_fp, "%i %i 1\n", (uid_t)target_uid, uid);
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
    }

    return(0);
}

int _singularity_runtime_ns_user_join(void) {
    int ns_fd = atoi(singularity_registry_get("DAEMON_NS_FD"));
    int user_fd;

    if ( singularity_priv_userns_enabled() ) {
        if ( ! singularity_daemon_has_namespace("user") ) {
            return(0);
        }

        /* Attempt to open /proc/[PID]/ns/user */
        user_fd = openat(ns_fd, "user", O_RDONLY);

        if( user_fd == -1 ) {
            singularity_message(ERROR, "Could not open USER NS fd: %s\n", strerror(errno));
            ABORT(255);
        }
    
        singularity_message(DEBUG, "Attempting to join USER namespace\n");
        if ( setns(user_fd, CLONE_NEWUSER) < 0 ) {
            singularity_message(ERROR, "Could not join USER namespace: %s\n", strerror(errno));
            ABORT(255);
        }
        singularity_message(DEBUG, "Successfully joined USER namespace\n");
    
        close(user_fd);
    }

    return(0);
}
