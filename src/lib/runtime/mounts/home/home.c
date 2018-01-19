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

#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <string.h>
#include <sys/mount.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <unistd.h>
#include <stdlib.h>

#include "config.h"
#include "util/file.h"
#include "util/util.h"
#include "util/message.h"
#include "util/privilege.h"
#include "util/config_parser.h"
#include "util/registry.h"
#include "util/mount.h"

#include "../../runtime.h"


int _singularity_runtime_mount_home(void) {
    char *home_source = singularity_priv_homedir();
    char *home_dest = singularity_priv_home();
    char *session_dir = singularity_registry_get("SESSIONDIR");
    char *container_dir = CONTAINER_FINALDIR;


    if ( singularity_config_get_bool(MOUNT_HOME) <= 0 ) {
        singularity_message(VERBOSE, "Skipping home dir mounting (per config)\n");
        return(0);
    }

    singularity_message(DEBUG, "Checking that home directry is configured: %s\n", home_dest);
    if ( home_dest == NULL ) {
        singularity_message(ERROR, "Could not obtain user's home directory\n");
        ABORT(255);
    }

    singularity_message(DEBUG, "Checking if home directories are being influenced by user\n");
    if ( singularity_registry_get("HOME") != NULL ) {
#ifndef SINGULARITY_NO_NEW_PRIVS
        singularity_message(WARNING, "Not mounting user requested home: host does not support PR_SET_NO_NEW_PRIVS\n");
        ABORT(255);
#endif
        singularity_message(DEBUG, "Checking if user bind control is allowed\n");
        if ( singularity_config_get_bool(USER_BIND_CONTROL) <= 0 ) {
            singularity_message(ERROR, "Not mounting user requested home: User bind control is disallowed\n");
            ABORT(255);
        }
    }

    singularity_message(DEBUG, "Checking ownership of home directory source: %s\n", home_source);
    if ( is_owner(home_source, singularity_priv_getuid()) != 0 ) {
        singularity_message(ERROR, "Home directory is not owned by calling user: %s\n", home_source);
        ABORT(255);
    }

    singularity_message(DEBUG, "Checking to make sure home directory destination is a full path: %s\n", home_dest);
    if ( home_dest[0] != '/' ) {
        singularity_message(ERROR, "Home directory must be a full path: %s\n", home_dest);
        ABORT(255);
    }

    singularity_message(DEBUG, "Checking if home directory is already mounted: %s\n", home_dest);
    if ( check_mounted(home_dest) >= 0 ) {
        singularity_message(VERBOSE, "Not mounting home directory (already mounted in container): %s\n", home_dest);
        return(0);
    }

    singularity_message(DEBUG, "Creating temporary directory to stage home: %s\n", joinpath(session_dir, home_dest));
    if ( s_mkpath(joinpath(session_dir, home_dest), 0755) < 0 ) {
        singularity_message(ERROR, "Failed creating home directory stage %s: %s\n", joinpath(session_dir, home_dest), strerror(errno));
        ABORT(255);
    }

    singularity_message(DEBUG, "Checking if SINGULARITY_CONTAIN is set\n");
    if ( ( singularity_registry_get("CONTAIN") == NULL ) || ( singularity_registry_get("HOME") != NULL ) ) {
        singularity_priv_escalate();
        singularity_message(VERBOSE, "Mounting home directory source into session directory: %s -> %s\n", home_source, joinpath(session_dir, home_dest));
        if ( singularity_mount(home_source, joinpath(session_dir, home_dest), NULL, MS_BIND | MS_NOSUID | MS_NODEV | MS_REC, NULL) < 0 ) {
            singularity_message(ERROR, "Failed to mount home directory %s -> %s: %s\n", home_source, joinpath(session_dir, home_dest), strerror(errno));
            ABORT(255);
        }
        if ( singularity_priv_userns_enabled() != 1 ) {
            if ( singularity_mount(NULL, joinpath(session_dir, home_dest), NULL, MS_BIND | MS_REMOUNT | MS_NODEV | MS_NOSUID | MS_REC, NULL) < 0 ) {
                singularity_message(ERROR, "Failed to remount home directory base %s: %s\n", joinpath(session_dir, home_dest), strerror(errno));
                ABORT(255);
            }
        }
        singularity_priv_drop();
    } else {
        singularity_message(VERBOSE, "Using sessiondir for home directory\n");
    }

    singularity_message(DEBUG, "Checking if overlay is enabled\n");
    if ( singularity_registry_get("OVERLAYFS_ENABLED") == NULL ) {
        char *homedir_base;

        singularity_message(DEBUG, "Staging home directory base\n");

        singularity_message(DEBUG, "Identifying the base home directory: %s\n", home_dest);
        if ( ( homedir_base = basedir(home_dest) ) == NULL ) {
            singularity_message(ERROR, "Could not identify base home directory path: %s\n", home_dest);
            ABORT(255);
        }

        singularity_message(DEBUG, "Checking home directory base exists in container: %s\n", homedir_base);
        if ( is_dir(joinpath(container_dir, homedir_base)) != 0 ) {
            singularity_message(ERROR, "Base home directory does not exist within the container: %s\n", homedir_base);
            ABORT(255);
        }

        singularity_priv_escalate();
        singularity_message(VERBOSE, "Mounting staged home directory base to container's base dir: %s -> %s\n", joinpath(session_dir, homedir_base), joinpath(container_dir, homedir_base));
        if ( singularity_mount(joinpath(session_dir, homedir_base), joinpath(container_dir, homedir_base), NULL, MS_BIND | MS_NOSUID | MS_NODEV | MS_REC, NULL) < 0 ) {
            singularity_message(ERROR, "Failed to mount staged home base: %s -> %s: %s\n", joinpath(session_dir, homedir_base), joinpath(container_dir, homedir_base), strerror(errno));
            ABORT(255);
        }
        singularity_priv_drop();

        free(homedir_base);
    } else {
        singularity_message(DEBUG, "Staging home directory\n");

        singularity_priv_escalate();
        singularity_message(DEBUG, "Creating home directory within container: %s\n", joinpath(container_dir, home_dest));
        if ( s_mkpath(joinpath(container_dir, home_dest), 0755) < 0 ) {
            singularity_message(ERROR, "Failed creating home directory in container %s: %s\n", joinpath(container_dir, home_dest), strerror(errno));
            ABORT(255);
        }

        singularity_message(VERBOSE, "Mounting staged home directory to container: %s -> %s\n", joinpath(session_dir, home_dest), joinpath(container_dir, home_dest));
        if ( singularity_mount(joinpath(session_dir, home_dest), joinpath(container_dir, home_dest), NULL, MS_BIND | MS_NOSUID | MS_NODEV | MS_REC, NULL) < 0 ) {
            singularity_message(ERROR, "Failed to mount staged home base: %s -> %s: %s\n", joinpath(session_dir, home_dest), joinpath(container_dir, home_dest), strerror(errno));
            ABORT(255);
        }
        singularity_priv_drop();

    }

    envar_set("HOME", home_dest, 1);

    free(home_source);
    free(home_dest);
    free(session_dir);

    return(0);
}
