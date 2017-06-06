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

#ifndef _GNU_SOURCE
#define _GNU_SOURCE
#endif

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
#include "util/message.h"
#include "util/privilege.h"
#include "util/config_parser.h"
#include "util/registry.h"

#include "../mount-util.h"
#include "../../runtime.h"


int _singularity_runtime_mount_cwd(void) {
    char *container_dir = singularity_runtime_rootfs(NULL);
    char *cwd_path = NULL;
    int r;

    singularity_message(DEBUG, "Checking to see if we should mount current working directory\n");

    singularity_message(DEBUG, "Getting current working directory\n");
    cwd_path = get_current_dir_name();
    if ( cwd_path == NULL ) {
        singularity_message(ERROR, "Could not obtain current directory path: %s\n", strerror(errno));
        ABORT(1);
    }

    singularity_message(DEBUG, "Checking if current directory exists in container\n");
    if ( is_dir(joinpath(container_dir, cwd_path)) == 0 ) {
        char *cwd_fileid = file_devino(cwd_path);
        char *container_cwd_fileid = file_devino(joinpath(container_dir, cwd_path));

        singularity_message(DEBUG, "Checking if container's cwd == host's cwd\n");
        if ( strcmp(cwd_fileid, container_cwd_fileid) == 0 ) {
            singularity_message(VERBOSE, "Not mounting current directory: location already available within container\n");
            free(cwd_path);
            free(cwd_fileid);
            free(container_cwd_fileid);
            return(0);
        } else {
            singularity_message(DEBUG, "Container's cwd is not the same as the host, continuing on...\n");
        }
    } else {
        singularity_message(DEBUG, "Container does not have the directory: %s\n", cwd_path);
    }

    singularity_message(DEBUG, "Checking for contain option\n");
    if ( singularity_registry_get("CONTAIN") != NULL ) {
        singularity_message(VERBOSE, "Not mounting current directory: contain was requested\n");
        free(cwd_path);
        return(0);
    }

    singularity_message(DEBUG, "Checking if CWD is already mounted: %s\n", cwd_path);
    if ( check_mounted(cwd_path) >= 0 ) {
        singularity_message(VERBOSE, "Not mounting CWD (already mounted in container): %s\n", cwd_path);
        free(cwd_path);
        return(0);
    }

    singularity_message(DEBUG, "Checking if cwd is in an operating system directory\n");
    if ( ( strcmp(cwd_path, "/") == 0 ) ||
         ( strcmp(cwd_path, "/bin") == 0 ) ||
         ( strcmp(cwd_path, "/etc") == 0 ) ||
         ( strcmp(cwd_path, "/mnt") == 0 ) ||
         ( strcmp(cwd_path, "/usr") == 0 ) ||
         ( strcmp(cwd_path, "/var") == 0 ) ||
         ( strcmp(cwd_path, "/opt") == 0 ) ||
         ( strcmp(cwd_path, "/sbin") == 0 ) ) {
        singularity_message(VERBOSE, "Not mounting CWD within operating system directory: %s\n", cwd_path);
        free(cwd_path);
        return(0);
    }

    singularity_message(DEBUG, "Checking if overlay is enabled\n");
    if ( singularity_registry_get("OVERLAYFS_ENABLED") == NULL ) {
        if ( is_dir(joinpath(container_dir, cwd_path)) < 0 ) {
            singularity_message(VERBOSE, "Not mounting current directory: overlay is not enabled and directory does not exist in container: %s\n", joinpath(container_dir, cwd_path));
            free(cwd_path);
            return(0);
        }
    }

    singularity_message(DEBUG, "Checking configuration file for 'user bind control'\n");
    if ( singularity_config_get_bool(USER_BIND_CONTROL) <= 0 ) {
        singularity_message(WARNING, "Not mounting current directory: user bind control is disabled by system administrator\n");
        free(cwd_path);
        return(0);
    }

#ifndef SINGULARITY_NO_NEW_PRIVS
    singularity_message(WARNING, "Not mounting current directory: host does not support PR_SET_NO_NEW_PRIVS\n");
    free(cwd_path);
    return(0);
#endif  

    singularity_message(DEBUG, "Creating current directory within container: %s\n", joinpath(container_dir, cwd_path));
    if ( s_mkpath(joinpath(container_dir, cwd_path), 0755) != 0 ) {
        singularity_priv_escalate();
        singularity_message(DEBUG, "Creating current directory (privileged) within container: %s\n", joinpath(container_dir, cwd_path));
        r = s_mkpath(joinpath(container_dir, cwd_path), 0755);
        singularity_priv_drop();
        if ( r < 0 ) {
            singularity_message(VERBOSE, "Could not create directory for current directory, skipping CWD mount\n");
            free(cwd_path);
            return(0);
        }
    }

    singularity_message(VERBOSE, "Binding '%s' to '%s/%s'\n", cwd_path, container_dir, cwd_path);
    singularity_priv_escalate();
    r = mount(cwd_path, joinpath(container_dir, cwd_path), NULL, MS_BIND|MS_NOSUID|MS_REC, NULL);
    if ( singularity_priv_userns_enabled() != 1 ) {
        r = mount(NULL, joinpath(container_dir, cwd_path), NULL, MS_BIND|MS_NOSUID|MS_REC|MS_REMOUNT, NULL);
    }
    singularity_priv_drop();
    if ( r < 0 ) {
        singularity_message(WARNING, "Could not bind CWD to container %s: %s\n", cwd_path, strerror(errno));
    }

    free(cwd_path);
    return(0);
}

