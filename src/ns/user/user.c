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


#include "file.h"
#include "util.h"
#include "message.h"
#include "config_parser.h"
#include "privilege.h"

static int userns_enabled = -1;


int singularity_ns_user_unshare(void) {

    if ( ( is_suid("/proc/self/exe") == 0 ) || ( getuid() == 0 ) ) {
        message(VERBOSE3, "Not virtualizing USER namespace: running privliged mode\n");
        return(0);
    }
#ifdef NS_CLONE_NEWUSER
    message(DEBUG, "Attempting to virtualize the USER namespace\n");
    if ( unshare(CLONE_NEWUSER) == 0 ) {
        message(DEBUG, "Enabled user namespaces\n");
        userns_enabled = 0;

        int child_pid = fork();

        if ( child_pid == 0 ) {
            // Allow the child to continue on, and catch the parent below

        } else if ( child_pid > 0 ) {
            int tmpstatus;
            int retval;

            uid_t uid = priv_getuid();
            gid_t gid = priv_getgid();

            {
                message(DEBUG, "Setting setgroups to: 'deny'\n");
                char *map_file = (char *) malloc(PATH_MAX);
                snprintf(map_file, PATH_MAX-1, "/proc/%d/setgroups", getpid());
                FILE *map_fp = fopen(map_file, "w+");
                if ( map_fp != NULL ) {
                    message(DEBUG, "Updating setgroups: %s\n", map_file);
                    fprintf(map_fp, "deny\n");
                    if ( fclose(map_fp) < 0 ) {
                        message(ERROR, "Failed to write deny to setgroup file %s: %s\n", map_file, strerror(errno));
                        ABORT(255);
                    }
                } else {
                    message(ERROR, "Could not write info to setgroups: %s\n", strerror(errno));
                    ABORT(255);
                }
                free(map_file);
            }
            {   
                message(DEBUG, "Setting GID map to: '0 %i 1'\n", gid);
                char *map_file = (char *) malloc(PATH_MAX);
                snprintf(map_file, PATH_MAX-1, "/proc/%d/gid_map", getpid());
                FILE *map_fp = fopen(map_file, "w+");
                if ( map_fp != NULL ) {
                    message(DEBUG, "Updating the parent gid_map: %s\n", map_file);
                    fprintf(map_fp, "0 %i 1\n", gid);
                    if ( fclose(map_fp) < 0 ) {
                        message(ERROR, "Failed to write to GID map %s: %s\n", map_file, strerror(errno));
                        ABORT(255);
                    }
                } else {
                    message(ERROR, "Could not write parent info to gid_map: %s\n", strerror(errno));
                    ABORT(255);
                }
                free(map_file);
            }
            {   
                message(DEBUG, "Setting UID map to: '0 %i 1'\n", uid);
                char *map_file = (char *) malloc(PATH_MAX);
                snprintf(map_file, PATH_MAX-1, "/proc/%d/uid_map", getpid());
                FILE *map_fp = fopen(map_file, "w+");
                if ( map_fp != NULL ) {
                    message(DEBUG, "Updating the parent uid_map: %s\n", map_file);
                    fprintf(map_fp, "0 %i 1\n", uid);
                    if ( fclose(map_fp) < 0 ) {
                        message(ERROR, "Failed to write to UID map %s: %s\n", map_file, strerror(errno));
                        ABORT(255);
                    }
                } else {
                    message(ERROR, "Could not write parent info to uid_map: %s\n", strerror(errno));
                    ABORT(255);
                }
                free(map_file);
            }

            message(DEBUG, "Waiting on NS child process\n");

            waitpid(child_pid, &tmpstatus, 0);
            retval = WEXITSTATUS(tmpstatus);
            exit(retval);
        } else {
            message(ERROR, "Failed forking child process\n");
            ABORT(255);
        }

    } else {
        message(VERBOSE3, "Not virtualizing USER namespace: runtime support failed\n");
    }
#else
    message(VERBOSE3, "Not virtualizing USER namespace: support not compiled in\n");
#endif

    return(0);
}


int singularity_ns_user_drop(void) {
    uid_t uid = priv_getuid();
    gid_t gid = priv_getgid();

    if ( userns_enabled < 0 ) {
        return(0);
    }

#ifdef NS_CLONE_NEWUSER

    if ( unshare(CLONE_NEWUSER) < 0 ) {
        message(ERROR, "Failed to unshare the user namespace within child: %s\n", strerror(errno));
        ABORT(255);
    }

    {
        message(DEBUG, "Setting setgroups to: 'deny'\n");
        char *map_file = (char *) malloc(PATH_MAX);
        snprintf(map_file, PATH_MAX-1, "/proc/%d/setgroups", getpid());
        FILE *map_fp = fopen(map_file, "w+");
        if ( map_fp != NULL ) {
            message(DEBUG, "Updating setgroups: %s\n", map_file);
            fprintf(map_fp, "deny\n");
            if ( fclose(map_fp) < 0 ) {
                message(ERROR, "Failed to write deny to setgroup file %s: %s\n", map_file, strerror(errno));
                ABORT(255);
            }
        } else {
            message(ERROR, "Could not write info to setgroups: %s\n", strerror(errno));
            ABORT(255);
        }
        free(map_file);
    }
    {   
        message(DEBUG, "Setting GID map to: '%i 0 1'\n", gid);
        char *map_file = (char *) malloc(PATH_MAX);
        snprintf(map_file, PATH_MAX-1, "/proc/%d/gid_map", getpid());
        FILE *map_fp = fopen(map_file, "w+");
        if ( map_fp != NULL ) {
            message(DEBUG, "Updating the parent gid_map: %s\n", map_file);
            fprintf(map_fp, "%i 0 1\n", gid);
            if ( fclose(map_fp) < 0 ) {
                message(ERROR, "Failed to write to GID map %s: %s\n", map_file, strerror(errno));
                ABORT(255);
            }
        } else {
            message(ERROR, "Could not write parent info to gid_map: %s\n", strerror(errno));
            ABORT(255);
        }
        free(map_file);
    }
    {   
        message(DEBUG, "Setting UID map to: '%i 0 1'\n", uid);
        char *map_file = (char *) malloc(PATH_MAX);
        snprintf(map_file, PATH_MAX-1, "/proc/%d/uid_map", getpid());
        FILE *map_fp = fopen(map_file, "w+");
        if ( map_fp != NULL ) {
            message(DEBUG, "Updating the parent uid_map: %s\n", map_file);
            fprintf(map_fp, "%i 0 1\n", uid);
            if ( fclose(map_fp) < 0 ) {
                message(ERROR, "Failed to write to UID map %s: %s\n", map_file, strerror(errno));
                ABORT(255);
            }
        } else {
            message(ERROR, "Could not write parent info to uid_map: %s\n", strerror(errno));
            ABORT(255);
        }
        free(map_file);
    }

#else
    message(VERBOSE3, "Not virtualizing USER namespace: support not compiled in\n");
#endif
    return(0);
}

int singularity_ns_user_enabled(void) {
    message(DEBUG, "Singularity user namespace enabled: %d\n", userns_enabled);
    return(userns_enabled);
}
