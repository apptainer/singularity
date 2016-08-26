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

#include "file.h"
#include "util.h"
#include "message.h"
#include "privilege.h"
#include "config_parser.h"
#include "rootfs/rootfs.h"
#include "ns/ns.h"

int singularity_mount_binds(void) {
    char *tmp_config_string;
    char *container_dir = singularity_rootfs_dir();

    if ( getenv("SINGULARITY_CONTAIN") != NULL ) {
        message(DEBUG, "Skipping bind mounts as contain was requested\n");
        return(0);
    }

    message(DEBUG, "Checking configuration file for 'bind path'\n");
    config_rewind();
    while ( ( tmp_config_string = config_get_key_value("bind path") ) != NULL ) {
        char *source = strtok(tmp_config_string, ",");
        char *dest = strtok(NULL, ",");
        chomp(source);
        if ( dest == NULL ) {
            dest = strdup(source);
        } else {
            if ( dest[0] == ' ' ) {
                dest++;
            }
            chomp(dest);
        }

        message(VERBOSE2, "Found 'bind path' = %s, %s\n", source, dest);

        if ( ( is_file(source) != 0 ) && ( is_dir(source) != 0 ) ) {
            message(WARNING, "Non existant 'bind path' source: '%s'\n", source);
            continue;
        }

        if ( ( is_file(source) == 0 ) && ( is_file(joinpath(container_dir, dest)) < 0 ) ) {
            if ( singularity_rootfs_overlay_enabled() > 0 ) {
                singularity_priv_escalate();
                FILE *tmp = fopen(joinpath(container_dir, dest), "w+");
                singularity_priv_drop();
                if ( fclose(tmp) != 0 ) {
                    message(WARNING, "Could not create bind point file in container %s: %s\n", dest, strerror(errno));
                    continue;
                }
            } else {
                message(WARNING, "Non existant bind point (file) in container: '%s'\n", dest);
                continue;
            }
        } else if ( ( is_dir(source) == 0 ) && ( is_dir(joinpath(container_dir, dest)) < 0 ) ) {
            if ( singularity_rootfs_overlay_enabled() > 0 ) {
                singularity_priv_escalate();
                if ( s_mkpath(joinpath(container_dir, dest), 0755) < 0 ) {
                    singularity_priv_drop();
                    message(WARNING, "Could not create bind point directory in container %s: %s\n", dest, strerror(errno));
                    continue;
                }
                singularity_priv_drop();
            } else {
                message(WARNING, "Non existant bind point (directory) in container: '%s'\n", dest);
                continue;
            }
        }

        singularity_priv_escalate();
        message(VERBOSE, "Binding '%s' to '%s/%s'\n", source, container_dir, dest);
        if ( mount(source, joinpath(container_dir, dest), NULL, MS_BIND|MS_NOSUID|MS_REC, NULL) < 0 ) {
            message(ERROR, "There was an error binding the path %s: %s\n", source, strerror(errno));
            ABORT(255);
        }
        singularity_priv_drop();
    }

    return(0);
}

