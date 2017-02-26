/* 
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

#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <string.h>
#include <sys/mount.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <unistd.h>
#include <stdlib.h>

#include "util/file.h"
#include "util/util.h"
#include "util/message.h"
#include "util/privilege.h"
#include "util/config_parser.h"
#include "util/registry.h"

#include "../mount-util.h"
#include "../../runtime.h"


int _singularity_runtime_mount_home(void) {
    char *homedir_base = NULL;
    char *homedir = singularity_priv_home();
    char *homedir_source = singularity_priv_homedir();
    char *container_dir = singularity_runtime_rootfs(NULL);
    char *tmpdir = singularity_runtime_tmpdir(NULL);

    if ( tmpdir == NULL ) {
        singularity_message(ERROR, "internal error - singularity_runtime_tmpdir() not set\n");
        ABORT(255);
    }

    if ( singularity_config_get_bool(MOUNT_HOME) <= 0 ) {
        singularity_message(VERBOSE, "Skipping home dir mounting (per config)\n");
        return(0);
    }

    singularity_message(DEBUG, "Checking home and home dir (%s, %s)\n", homedir, homedir_source);
    if ( ( homedir == NULL ) || ( homedir_source == NULL ) ) {
        singularity_message(ERROR, "Could not obtain user's home directory\n");
        ABORT(255);
    }

    singularity_message(DEBUG, "Identifying the base directory of homedir: %s\n", homedir);
    if ( ( homedir_base = basedir(homedir) ) == NULL ) {
        singularity_message(ERROR, "Could not identify basedir for home directory path: %s\n", homedir);
        ABORT(255);
    }

    singularity_message(DEBUG, "Checking if home directory is already mounted: %s\n", homedir);
    if ( check_mounted(homedir) >= 0 ) {
        singularity_message(VERBOSE, "Not mounting home directory (already mounted in container): %s\n", homedir);
        return(0);
    }

    singularity_message(DEBUG, "Creating directory to stage tmpdir home: %s\n", joinpath(tmpdir, homedir));
    if ( s_mkpath(joinpath(tmpdir, homedir), 0755) < 0 ) {
        singularity_message(ERROR, "Failed creating home directory stage\n");
    }

    if ( is_dir(homedir_base) != 0 ) {
        singularity_message(DEBUG, "Trying to create base home dir within container: %s\n", homedir_base);
        if ( singularity_registry_get("OVERLAYFS_ENABLED") != NULL ) {
            singularity_priv_escalate();
            if ( s_mkpath(joinpath(container_dir, homedir_base), 0755) == 0 ) {
                singularity_priv_drop();
                singularity_message(DEBUG, "Created home directory within the container: %s\n", homedir_base);
            } else {
                singularity_priv_drop();
            }
        } else {
            singularity_message(ERROR, "Base home directory does not exist within the container: %s\n", homedir_base);
            ABORT(255);

        }
    }

    if ( ( singularity_registry_get("CONTAIN") != NULL ) && ( singularity_registry_get("HOME") == NULL ) ) {
        singularity_message(VERBOSE, "Requested --contain option, using temporary storage for home directory\n");
    } else {
        singularity_priv_escalate();
        singularity_message(VERBOSE, "Mounting home directory source to stage: %s->%s\n", homedir_source, joinpath(tmpdir, homedir));
        if ( mount(homedir_source, joinpath(tmpdir, homedir), NULL, MS_BIND | MS_REC, NULL) < 0 ) {
            singularity_message(ERROR, "Failed to mount home directory %s: %s\n", homedir_source, strerror(errno));
            ABORT(255);
        }
        singularity_priv_drop();
    }

    singularity_priv_escalate();
    if ( singularity_priv_userns_enabled() != 1 ) {
        singularity_message(DEBUG, "Remounting home directory with necessary options: %s\n", homedir);
        if ( mount(NULL, joinpath(tmpdir, homedir), NULL, MS_BIND | MS_REMOUNT | MS_NODEV | MS_NOSUID | MS_REC , NULL) < 0 ) {
            singularity_message(ERROR, "Failed to remount home directory %s: %s\n", homedir, strerror(errno));
            ABORT(255);
        }
    }
    singularity_message(VERBOSE, "Mounting home directory base into container: %s->%s\n", joinpath(tmpdir, homedir_base), joinpath(container_dir, homedir_base));
    if ( mount(joinpath(tmpdir, homedir_base), joinpath(container_dir, homedir_base), NULL, MS_BIND | MS_REC, NULL) < 0 ) {
        singularity_message(ERROR, "Failed to mount home directory base %s: %s\n", homedir_base, strerror(errno));
        ABORT(255);
    }
    if ( singularity_priv_userns_enabled() != 1 ) {
        if ( mount(NULL, joinpath(container_dir, homedir_base), NULL, MS_BIND | MS_REMOUNT | MS_NODEV | MS_NOSUID | MS_REC, NULL) < 0 ) {
            singularity_message(ERROR, "Failed to remount home directory base %s: %s\n", homedir_base, strerror(errno));
            ABORT(255);
        }
    }
    singularity_priv_drop();

    return(0);
}
