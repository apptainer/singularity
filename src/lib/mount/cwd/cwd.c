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

#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <string.h>
#include <sys/mount.h>
#include <sys/stat.h>
#include <unistd.h>
#include <stdlib.h>
#include <libgen.h>
#include <linux/limits.h>

#include "util/file.h"
#include "util/util.h"
#include "lib/message.h"
#include "lib/privilege.h"
#include "lib/config_parser.h"
#include "lib/rootfs/rootfs.h"
#include "lib/ns/ns.h"
#include "../mount-util.h"


void singularity_mount_cwd(void) {
    char *container_dir = singularity_rootfs_dir();
    char *cwd_path = NULL;
    char *cwd_fileid = NULL;
    char *container_cwd_fileid = NULL;
    int r;
    int user_bind_control;

    singularity_message(DEBUG, "Checking to see if we should mount current working directory\n");
    
#ifndef SINGULARITY_NO_NEW_PRIVS
    singularity_message(WARNING, "Not mounting current directory: host does not support PR_SET_NO_NEW_PRIVS\n");
    return;
#endif
    
    singularity_message(DEBUG, "Checking for contain option\n");
    if ( envar_defined("SINGULARITY_CONTAIN") == TRUE ) {
        singularity_message(VERBOSE, "Not mounting current directory: contain was requested\n");
        return;
    }

    singularity_message(DEBUG, "Checking configuration file for 'user bind control'\n");
    singularity_config_rewind();
    if ( (user_bind_control = singularity_config_get_bool("user bind control", 1)) <= 0 ) {
        singularity_message(DEBUG, "User bind control disabled by system administrator\n");
    }

    singularity_message(DEBUG, "Getting current working directory\n");
    cwd_path = get_current_dir_name();
    if ( cwd_path == NULL ) {
        singularity_message(ERROR, "Could not obtain current directory path: %s\n", strerror(errno));
        ABORT(1);
    }    
    
    singularity_message(DEBUG, "Checking if current directory exists in container\n");
    if ( is_dir(joinpath(container_dir, cwd_path)) == 0 ) {
        cwd_fileid = file_devino(cwd_path);
        container_cwd_fileid = file_devino(joinpath(container_dir, cwd_path));

        singularity_message(DEBUG, "Checking if container's cwd == host's cwd\n");
        if ( (check_mounted(cwd_path) >= 0) || (strcmp(cwd_fileid, container_cwd_fileid) == 0) ) {
            singularity_message(VERBOSE, "Not mounting current directory: location already available within container\n");
            free(cwd_path);
            free(container_dir);
            free(cwd_fileid);
            free(container_cwd_fileid);
            return;
        } else {
            free(cwd_fileid);
            free(container_cwd_fileid);
            if ( user_bind_control == 1 ) {
                singularity_message(DEBUG, "Working directory exists in container but is not already mounted, continuing on...\n");
            } else {
                singularity_message(WARNING, "Not mounting current directory: user bind control is disabled by system administrator\n");
                free(cwd_path);
                free(container_dir);
                return;
            }
        }
    } else {
        if ( user_bind_control == 1 ) {
            singularity_message(DEBUG, "Container does not have the directory: %s\n", cwd_path);
            singularity_message(DEBUG, "Checking if overlay is enabled\n");
            if ( singularity_rootfs_overlay_enabled() <= 0 ) {
                singularity_message(VERBOSE, "Not mounting current directory: overlay is not enabled and directory does not exist in container: %s\n", joinpath(container_dir, cwd_path));
                free(cwd_path);
                free(container_dir);
                return;
            } else {
                singularity_message(DEBUG, "Overlay is enabled: attempting to create current working directory inside container\n");
                singularity_priv_escalate();
                r =  s_mkpath(joinpath(container_dir, cwd_path), 0755);
                singularity_priv_drop();
                if ( r < 0 ) {
                    singularity_message(VERBOSE, "Could not create directory for current directory, skipping CWD mount\n");
                    free(cwd_path);
                    return;
                }
            }
        } else {
            singularity_message(WARNING, "Not mounting current directory: user bind control is disabled by system administrator\n");
            return;
        }
    }
    
    singularity_priv_escalate();
    singularity_message(VERBOSE, "Binding '%s' to '%s/%s'\n", cwd_path, container_dir, cwd_path);
    r = mount(cwd_path, joinpath(container_dir, cwd_path), NULL, MS_BIND|MS_NOSUID|MS_REC, NULL);
    singularity_priv_drop();
    if ( r < 0 ) {
        singularity_message(WARNING, "Could not bind CWD to container %s: %s\n", cwd_path, strerror(errno));
    }

    free(cwd_path);
    free(container_dir);
    return;
}

