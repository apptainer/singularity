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
#include <sys/types.h>
#include <unistd.h>
#include <stdlib.h>
#include <pwd.h>

#include "util/file.h"
#include "util/util.h"
#include "lib/message.h"
#include "lib/privilege.h"
#include "lib/config_parser.h"
#include "lib/sessiondir.h"
#include "lib/rootfs/rootfs.h"
#include "../mount-util.h"


int singularity_mount_home(void) {
    char *homedir;
    char *homedir_source;
    char *homedir_base = NULL;
    char *container_dir = singularity_rootfs_dir();
    char *sessiondir = singularity_sessiondir_get();
    struct passwd *pw;
    uid_t uid = singularity_priv_getuid();

    singularity_config_rewind();
    if ( singularity_config_get_bool("mount home", 1) <= 0 ) {
        singularity_message(VERBOSE, "Skipping tmp dir mounting (per config)\n");
        return(0);
    }

    errno = 0;
    if ( ( pw = getpwuid(uid) ) == NULL ) {
        // List of potential error codes for unknown name taken from man page.
        if ( (errno == 0) || (errno == ESRCH) || (errno == EBADF) || (errno == EPERM) ) {
            singularity_message(VERBOSE3, "Not mounting home directory as passwd entry for %d not found.\n", uid);
            return(1);
        } else {
            singularity_message(ERROR, "Failed to lookup username for UID %d: %s\n", getuid, strerror(errno));
            ABORT(255);
        }
    }

    singularity_message(DEBUG, "Obtaining user's homedir\n");
    homedir = pw->pw_dir;

    // Figure out home directory source
    if ( ( homedir_source = envar_path("SINGULARITY_HOME") ) != NULL ) {
        char *colon;
        singularity_config_rewind();
        if ( singularity_config_get_bool("user bind control", 1) <= 0 ) {
            singularity_message(ERROR, "User bind control is disabled by system administrator\n");
            ABORT(5);
        }

        colon = strchr(homedir_source, ':');
        if ( colon != NULL ) {
            homedir = colon + 1;
            *colon = '\0';
            singularity_message(VERBOSE2, "Set the home directory (via envar) to: %s\n", homedir);
        }

        singularity_message(VERBOSE2, "Set the home directory source (via envar) to: %s\n", homedir_source);
    } else if ( envar_defined("SINGULARITY_CONTAIN") == TRUE ) {
        char *tmpdirpath;
        if ( ( tmpdirpath = envar_path("SINGULARITY_WORKDIR")) != NULL ) {
            singularity_config_rewind();
            if ( singularity_config_get_bool("user bind control", 1) <= 0 ) {
                singularity_message(ERROR, "User bind control is disabled by system administrator\n");
                ABORT(5);
            }

            homedir_source = joinpath(tmpdirpath, "/home");
        } else {
            // TODO: Randomize tmp_home, so multiple calls to the same container don't overlap
            homedir_source = joinpath(sessiondir, "/home.tmp");
        }
        if ( s_mkpath(homedir_source, 0755) < 0 ) {
            singularity_message(ERROR, "Could not create temporary home directory %s: %s\n", homedir_source, strerror(errno));
            ABORT(255);
        } else {
            singularity_message(VERBOSE2, "Set the contained home directory source to: %s\n", homedir_source);
        }

        free(tmpdirpath);

    } else if ( is_dir(homedir) == 0 ) {
        homedir_source = strdup(homedir);
        singularity_message(VERBOSE2, "Set base the home directory source to: %s\n", homedir_source);
    } else {
        singularity_message(ERROR, "Could not identify home directory path: %s\n", homedir_source);
        ABORT(255);
    }

    singularity_message(DEBUG, "Checking if home directory is already mounted: %s\n", homedir);
    if ( check_mounted(homedir) >= 0 ) {
        singularity_message(VERBOSE, "Not mounting home directory (already mounted in container): %s\n", homedir);
        return(0);
    }

    // Create a location to stage the directories
    if ( s_mkpath(homedir_source, 0755) < 0 ) {
        singularity_message(ERROR, "Failed creating home directory bind path\n");
    }

    // Create a location to stage the directories
    if ( s_mkpath(joinpath(sessiondir, homedir), 0755) < 0 ) {
        singularity_message(ERROR, "Failed creating home directory bind path\n");
    }

    // Check to make sure whatever we were given as the home directory is really ours
    singularity_message(DEBUG, "Checking permissions on home directory: %s\n", homedir_source);
    if ( is_owner(homedir_source, uid) < 0 ) {
        singularity_message(ERROR, "Home directory ownership incorrect: %s\n", homedir_source);
        ABORT(255);
    }

    // Figure out where we should mount the home directory in the container
    singularity_message(DEBUG, "Trying to create home dir within container\n");
    if ( singularity_rootfs_overlay_enabled() > 0 ) {
        singularity_priv_escalate();
        if ( s_mkpath(joinpath(container_dir, homedir), 0755) == 0 ) {
            singularity_priv_drop();
            singularity_message(DEBUG, "Created home directory within the container: %s\n", homedir);
            homedir_base = strdup(homedir);
        } else {
            singularity_priv_drop();
        }
    }

    if ( homedir_base == NULL ) {
        if ( ( homedir_base = basedir(homedir) ) == NULL ) {
            singularity_message(ERROR, "Could not identify basedir for home directory path: %s\n", homedir);
        }
        if ( is_dir(joinpath(container_dir, homedir_base)) < 0 ) {
            singularity_message(WARNING, "Not mounting home directory: bind point does not exist in container: %s\n", homedir_base);
            return(1);
        }
    }

    singularity_priv_escalate();
    // First mount the real home directory to the stage
    singularity_message(VERBOSE, "Mounting home directory to stage: %s->%s\n", homedir_source, joinpath(sessiondir, homedir));
    if ( mount(homedir_source, joinpath(sessiondir, homedir), NULL, MS_BIND|MS_NOSUID|MS_REC, NULL) < 0 ) {
        singularity_message(ERROR, "Failed to mount home directory to stage: %s\n", strerror(errno));
        ABORT(255);
    }
    // Then mount the stage to the container
    singularity_message(VERBOSE, "Mounting staged home directory into container: %s->%s\n", joinpath(sessiondir, homedir_base), joinpath(container_dir, homedir_base));
    if ( mount(joinpath(sessiondir, homedir_base), joinpath(container_dir, homedir_base), NULL, MS_BIND|MS_NOSUID|MS_REC, NULL) < 0 ) {
        singularity_message(ERROR, "Failed to mount staged home directory into container: %s\n", strerror(errno));
        ABORT(255);
    }
    singularity_priv_drop();

    free(homedir_source);

    return(0);
}
