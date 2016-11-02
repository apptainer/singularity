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

#include "util/file.h"
#include "util/util.h"
#include "lib/message.h"
#include "lib/privilege.h"
#include "lib/config_parser.h"
#include "lib/rootfs/rootfs.h"
#include "lib/ns/ns.h"
#include "../mount-util.h"

int singularity_mount_binds(void) {
    char *tmp_config_string;
    char *container_dir = singularity_rootfs_dir();

    if ( envar_defined("SINGULARITY_CONTAIN") == TRUE ) {
        singularity_message(DEBUG, "Skipping bind mounts as contain was requested\n");
        return(0);
    }

    singularity_message(DEBUG, "Checking configuration file for 'bind path'\n");
    singularity_config_rewind();
    while ( ( tmp_config_string = singularity_config_get_value("bind path") ) != NULL ) {
        char *source = strtok(tmp_config_string, ":");
        char *dest = strtok(NULL, ":");
        chomp(source);
        if ( dest == NULL ) {
            dest = strdup(source);
        } else {
            chomp(dest);
        }

        singularity_message(VERBOSE2, "Found 'bind path' = %s, %s\n", source, dest);

        if ( ( is_file(source) < 0 ) && ( is_dir(source) < 0 ) ) {
            singularity_message(WARNING, "Non existant 'bind path' source: '%s'\n", source);
            continue;
        }

        singularity_message(DEBUG, "Checking if bind point is already mounted: %s\n", dest);
        if ( check_mounted(dest) >= 0 ) {
            singularity_message(VERBOSE, "Not mounting bind point (already mounted): %s\n", dest);
            continue;
        }

        if ( ( is_file(source) == 0 ) && ( is_file(joinpath(container_dir, dest)) < 0 ) ) {
            if ( singularity_rootfs_overlay_enabled() > 0 ) {
                singularity_priv_escalate();
                singularity_message(VERBOSE3, "Creating bind file on overlay file system: %s\n", dest);
                FILE *tmp = fopen(joinpath(container_dir, dest), "w+"); // Flawfinder: ignore
                singularity_priv_drop();
                if ( tmp == NULL ) {
                    singularity_message(WARNING, "Could not create bind point file in container %s: %s\n", dest, strerror(errno));
                    continue;
                }
                if ( fclose(tmp) != 0 ) {
                    singularity_message(WARNING, "Could not close bind point file descriptor %s: %s\n", dest, strerror(errno));
                    continue;
                }
                singularity_message(DEBUG, "Created bind file: %s\n", dest);
            } else {
                singularity_message(WARNING, "Non existant bind point (file) in container: '%s'\n", dest);
                continue;
            }
        } else if ( ( is_dir(source) == 0 ) && ( is_dir(joinpath(container_dir, dest)) < 0 ) ) {
            if ( singularity_rootfs_overlay_enabled() > 0 ) {
                singularity_priv_escalate();
                singularity_message(VERBOSE3, "Creating bind directory on overlay file system: %s\n", dest);
                if ( s_mkpath(joinpath(container_dir, dest), 0755) < 0 ) {
                    singularity_priv_drop();
                    singularity_message(WARNING, "Could not create bind point directory in container %s: %s\n", dest, strerror(errno));
                    continue;
                }
                singularity_priv_drop();
            } else {
                singularity_message(WARNING, "Non existant bind point (directory) in container: '%s'\n", dest);
                continue;
            }
        }

        singularity_priv_escalate();
        singularity_message(VERBOSE, "Binding '%s' to '%s/%s'\n", source, container_dir, dest);
        if ( mount(source, joinpath(container_dir, dest), NULL, MS_BIND|MS_NOSUID|MS_REC, NULL) < 0 ) {
            singularity_message(ERROR, "There was an error binding the path %s: %s\n", source, strerror(errno));
            ABORT(255);
        }
        singularity_priv_drop();
    }

    return(0);
}

