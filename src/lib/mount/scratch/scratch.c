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
#include <sys/wait.h>
#include <unistd.h>
#include <stdlib.h>
#include <libgen.h>
#include <linux/limits.h>

#include "util/file.h"
#include "util/util.h"
#include "lib/message.h"
#include "lib/privilege.h"
#include "lib/sessiondir.h"
#include "lib/fork.h"
#include "lib/config_parser.h"
#include "lib/rootfs/rootfs.h"
#include "lib/ns/ns.h"
#include "../mount-util.h"


void singularity_mount_scratch(void) {
    char *container_dir = singularity_rootfs_dir();
    char *scratchdir_path;
    char *tmpdir_path;
    char *sourcedir_path;
    int r;

    singularity_message(DEBUG, "Checking to see if we should mount scratch directory\n");

    singularity_message(DEBUG, "Checking configuration file for 'user bind control'\n");
    singularity_config_rewind();
    if ( singularity_config_get_bool("user bind control", 1) <= 0 ) {
        singularity_message(VERBOSE, "Not mounting current direcotry: user bind control is disabled by system administrator\n");
        return;
    }

    singularity_message(DEBUG, "Getting SINGULARITY_SCRATCHDIR from environment\n");
    if ( ( scratchdir_path = envar_path("SINGULARITY_SCRATCHDIR") ) == NULL ) {
        singularity_message(DEBUG, "Not mounting scratch directory: Not requested\n");
        return;
    }

    singularity_message(DEBUG, "Checking if overlay is enabled\n");
    int overlayfs_enabled = singularity_rootfs_overlay_enabled() > 0;
    if ( !overlayfs_enabled ) {
        singularity_message(VERBOSE, "Overlay is not enabled: cannot make directories not preexisting in container scratch.\n");
    }

    singularity_message(DEBUG, "Checking SINGULARITY_WORKDIR from environment\n");
    if ( ( tmpdir_path = envar_path("SINGULARITY_WORKDIR") ) != NULL ) {
//TODO: This should be done somewhere globally... Perhaps another module? YES!
//      But until then, we are counting on something else to clean us up (e.g.
//      the resource manager)
//
//        pid_t child_pid;
//        if ( ( child_pid = singularity_fork() ) > 0 ) {
//            int tmpstatus;
//            int retval;
//
//            singularity_message(DEBUG, "Cleanup thread waiting on child...\n");
//
//            waitpid(child_pid, &tmpstatus, 0);
//            retval = WEXITSTATUS(tmpstatus);
//
//            singularity_message(DEBUG, "Checking to see if we are the last process running in this sessiondir\n");
//            if ( flock(sessiondir_fd, LOCK_EX | LOCK_NB) == 0 ) {
//                singularity_message(VERBOSE, "Cleaning sessiondir: %s\n", sessiondir);
//                if ( s_rmdir(sessiondir) < 0 ) {
//                    singularity_message(ERROR, "Could not remove session directory %s: %s\n", sessiondir, strerror(errno));
//                }
//            }
//
//            exit(retval);
//        }

    } else {
        if ( ( tmpdir_path = singularity_sessiondir_get() ) == NULL ) {
            singularity_message(ERROR, "Could not identify a suitable temporary directory for scratch\n");
            return;
        }
    }

    sourcedir_path = joinpath(tmpdir_path, "/scratch");

    if ( s_mkpath(sourcedir_path, 0750) < 0 ) {
        singularity_message(ERROR, "Could not create scratch source directory %s: %s\n", sourcedir_path, strerror(errno));
        ABORT(255);
    }

    if (overlayfs_enabled) {
        singularity_priv_escalate();
        singularity_message(DEBUG, "Creating scratch directory inside container\n");
        r = s_mkpath(joinpath(container_dir, scratchdir_path), 0755);
        singularity_priv_drop();
        if ( r < 0 ) {
            singularity_message(VERBOSE, "Skipping scratch directory mount, could not create dir inside container %s: %s\n", scratchdir_path, strerror(errno));
            return;
        }
    }

    singularity_priv_escalate();
    singularity_message(VERBOSE, "Binding '%s' to '%s/%s'\n", sourcedir_path, container_dir, scratchdir_path);
    r = mount(sourcedir_path, joinpath(container_dir, scratchdir_path), NULL, MS_BIND|MS_NOSUID|MS_REC, NULL);
    singularity_priv_drop();
    if ( r < 0 ) {
        singularity_message(WARNING, "Could not bind scratch directory into container %s: %s\n", sourcedir_path, strerror(errno));
        ABORT(255);
    }

    return;
}

