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
    char *homedir = NULL;
    char *homedir_base = NULL;
    char *homedir_source = NULL;
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

    singularity_message(DEBUG, "Obtaining user's homedir\n");

    // Figure out home directory source
    if ( ( homedir = singularity_registry_get("HOME") ) != NULL ) {
        if ( singularity_priv_getuid() == 0 ) {
            singularity_message(ERROR, "Will not virtulize the root user's home directory\n");
            ABORT(1);
        }
        singularity_message(VERBOSE2, "Set the home directory source (via envar) to: %s\n", homedir);
    } else if ( ( homedir = singularity_priv_home() ) != NULL ) {
        singularity_message(VERBOSE2, "Set the home directory source (via getpwuid()) to: %s\n", homedir);
    } else {
        singularity_message(ERROR, "Could not obtain user's home directory\n");
        ABORT(255);
    }

    if ( singularity_registry_get("CONTAIN") == NULL ) {
        homedir_source = strdup(homedir);
    } else {
        char *tmpdirpath;
        if ( ( tmpdirpath = singularity_registry_get("WORKDIR") ) != NULL ) {
            if ( singularity_config_get_bool(USER_BIND_CONTROL) <= 0 ) {
                singularity_message(ERROR, "User bind control is disabled by system administrator\n");
                ABORT(5);
            }

            homedir_source = joinpath(tmpdirpath, "/home");
        } else {
            // TODO: Randomize tmp_home, so multiple calls to the same container don't overlap
            homedir_source = joinpath(tmpdir, "/home");
        }
        if ( s_mkpath(homedir_source, 0755) < 0 ) {
            singularity_message(ERROR, "Could not create temporary home directory %s: %s\n", homedir_source, strerror(errno));
            ABORT(255);
        } else {
            singularity_message(VERBOSE2, "Set the contained home directory source to: %s\n", homedir_source);
        }

        free(tmpdirpath);
    }

    if ( ( homedir_base = basedir(homedir) ) == NULL ) {
        singularity_message(ERROR, "Could not identify basedir for home directory path: %s\n", homedir);
    }

    singularity_message(DEBUG, "Checking if home directory is already mounted: %s\n", homedir);
    if ( check_mounted(homedir) >= 0 ) {
        singularity_message(VERBOSE, "Not mounting home directory (already mounted in container): %s\n", homedir);
        return(0);
    }

    // Create a location to stage the directories
    singularity_message(DEBUG, "Creating directory to stage home: %s\n", homedir_source);
    if ( s_mkpath(homedir_source, 0755) < 0 ) {
        singularity_message(ERROR, "Failed creating home directory bind path\n");
    }

    // Create a location to stage the directories
    singularity_message(DEBUG, "Creating directory to stage tmpdir home: %s\n", joinpath(tmpdir, homedir));
    if ( s_mkpath(joinpath(tmpdir, homedir), 0755) < 0 ) {
        singularity_message(ERROR, "Failed creating home directory bind path\n");
    }

    // Check to make sure whatever we were given as the home directory is really ours
    singularity_message(DEBUG, "Checking permissions on home directory: %s\n", homedir_source);
    if ( is_owner(homedir_source, singularity_priv_getuid()) < 0 ) {
        singularity_message(ERROR, "Home directory ownership incorrect: %s\n", homedir_source);
        ABORT(255);
    }

    // Figure out where we should mount the home directory in the container
    singularity_message(DEBUG, "Trying to create home dir within container\n");
    if ( singularity_registry_get("OVERLAYFS_ENABLED") != NULL ) {
        singularity_priv_escalate();
        if ( s_mkpath(joinpath(container_dir, homedir), 0755) == 0 ) {
            singularity_priv_drop();
            singularity_message(DEBUG, "Created home directory within the container: %s\n", homedir);
            homedir_base = strdup(homedir);
        } else {
            singularity_priv_drop();
        }
    }

    if ( is_dir(joinpath(container_dir, homedir_base)) < 0 ) {
        singularity_message(WARNING, "Not mounting home directory: bind point does not exist in container: %s\n", homedir_base);
        return(1);
    }

    singularity_priv_escalate();
    // First mount the real home directory to the stage
    singularity_message(VERBOSE, "Mounting home directory to stage: %s->%s\n", homedir_source, joinpath(tmpdir, homedir));
    if ( mount(homedir_source, joinpath(tmpdir, homedir), NULL, MS_BIND|MS_NOSUID|MS_REC, NULL) < 0 ) {
        singularity_message(ERROR, "Failed to mount home directory to stage: %s\n", strerror(errno));
        ABORT(255);
    }
    if ( singularity_priv_userns_enabled() != 1 ) {
        if ( mount(NULL, joinpath(tmpdir, homedir), NULL, MS_BIND|MS_NOSUID|MS_REC|MS_REMOUNT, NULL) < 0 ) {
            singularity_message(ERROR, "Failed to remount home directory to stage: %s\n", strerror(errno));
            ABORT(255);
        }
    }
    // Then mount the stage to the container
    singularity_message(VERBOSE, "Mounting staged home directory into container: %s->%s\n", joinpath(tmpdir, homedir_base), joinpath(container_dir, homedir_base));
    if ( mount(joinpath(tmpdir, homedir_base), joinpath(container_dir, homedir_base), NULL, MS_BIND|MS_NOSUID|MS_REC, NULL) < 0 ) {
        singularity_message(ERROR, "Failed to mount staged home directory into container: %s\n", strerror(errno));
        ABORT(255);
    }
    if ( singularity_priv_userns_enabled() != 1 ) {
        if ( mount(NULL, joinpath(container_dir, homedir_base), NULL, MS_BIND|MS_NOSUID|MS_REC|MS_REMOUNT, NULL) < 0 ) {
            singularity_message(ERROR, "Failed to remount staged home directory into container: %s\n", strerror(errno));
            ABORT(255);
        }
    }
    singularity_priv_drop();

    free(homedir_source);

    return(0);
}
