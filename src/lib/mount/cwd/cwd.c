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
    char *cwd_path = (char *) malloc(PATH_MAX);
    int r;

    singularity_message(DEBUG, "Checking to see if we should mount current working directory\n");

    singularity_message(DEBUG, "Getting current working directory\n");
    if ( getcwd(cwd_path, PATH_MAX) == NULL ) {
        singularity_message(ERROR, "Could not obtain current directory path: %s\n", strerror(errno));
        ABORT(1);
    }

    singularity_message(DEBUG, "Checking configuration file for 'user bind control'\n");
    singularity_config_rewind();
    if ( singularity_config_get_bool("user bind control", 1) <= 0 ) {
        singularity_message(WARNING, "Not mounting current directory: user bind control is disabled by system administrator\n");
        return;
    }

#ifndef SINGULARITY_NO_NEW_PRIVS
        singularity_message(WARNING, "Not mounting current directory: host does not support PR_SET_NO_NEW_PRIVS\n");
        return;
#endif  

    singularity_message(DEBUG, "Checking for contain option\n");
    if ( envar_defined("SINGULARITY_CONTAIN") == TRUE ) {
        singularity_message(VERBOSE, "Not mounting current directory: contain was requested\n");
        return;
    }

    singularity_message(DEBUG, "Checking if CWD is already mounted: %s\n", cwd_path);
    if ( check_mounted(cwd_path) >= 0 ) {
        singularity_message(VERBOSE, "Not mounting CWD (already mounted in container): %s\n", cwd_path);
        return;
    }

    singularity_message(DEBUG, "Checking if overlay is enabled\n");
    if ( singularity_rootfs_overlay_enabled() <= 0 ) {
        singularity_message(VERBOSE, "Not mounting current directory: overlay is not enabled\n");
        return;
    }

    singularity_priv_escalate();
    singularity_message(DEBUG, "Creating current working directory inside container\n");
    r = s_mkpath(joinpath(container_dir, cwd_path), 0755);
    singularity_priv_drop();
    if ( r < 0 ) {
        singularity_message(VERBOSE, "Could not create directory for current directory, skipping CWD mount\n");
        return;
    }

    singularity_priv_escalate();
    singularity_message(VERBOSE, "Binding '%s' to '%s/%s'\n", cwd_path, container_dir, cwd_path);
    r = mount(cwd_path, joinpath(container_dir, cwd_path), NULL, MS_BIND|MS_NOSUID|MS_REC, NULL);
    singularity_priv_drop();
    if ( r < 0 ) {
        singularity_message(WARNING, "Could not bind CWD to container %s: %s\n", cwd_path, strerror(errno));
    }

    return;
}

