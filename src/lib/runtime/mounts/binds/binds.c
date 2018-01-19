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
#include <unistd.h>
#include <stdlib.h>
#include <libgen.h>

#include "config.h"
#include "util/file.h"
#include "util/util.h"
#include "util/message.h"
#include "util/privilege.h"
#include "util/config_parser.h"
#include "util/registry.h"
#include "util/mount.h"

#include "../../runtime.h"


int _singularity_runtime_mount_binds(void) {
    char *tmp_config_string;
    char *container_dir = CONTAINER_FINALDIR;

    if ( singularity_registry_get("CONTAIN") != NULL ) {
        singularity_message(DEBUG, "Skipping bind mounts as contain was requested\n");
        return(0);
    }

    singularity_message(DEBUG, "Checking configuration file for 'bind path'\n");
    const char **tmp_config_string_list = singularity_config_get_value_multi(BIND_PATH);
    if ( strlength(*tmp_config_string_list, 1) == 0 ) {
        return(0);
    }
    while ( *tmp_config_string_list != NULL ) {
        tmp_config_string = strdup(*tmp_config_string_list);
        tmp_config_string_list++;
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
            singularity_message(WARNING, "Non existent 'bind path' source: '%s'\n", source);
            continue;
        }

        singularity_message(DEBUG, "Checking if bind point is already mounted: %s\n", dest);
        if ( check_mounted(dest) >= 0 ) {
            singularity_message(VERBOSE, "Not mounting bind point (already mounted): %s\n", dest);
            continue;
        }

        if ( ( is_file(source) == 0 ) && ( is_file(joinpath(container_dir, dest)) < 0 ) ) {
            if ( singularity_registry_get("OVERLAYFS_ENABLED") != NULL ) {
                char *basedir = dirname(joinpath(container_dir, dest));

                singularity_message(DEBUG, "Checking base directory for file %s ('%s')\n", dest, basedir);
                if ( is_dir(basedir) != 0 ) {
                    singularity_message(DEBUG, "Creating base directory for file bind\n");
                    singularity_priv_escalate();
                    if ( s_mkpath(basedir, 0755) != 0 ) {
                        singularity_message(ERROR, "Failed creating base directory to bind file: %s\n", dest);
                        ABORT(255);
                    }
                    singularity_priv_drop();
                }

                free(basedir);

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
                singularity_message(WARNING, "Non existent bind point (file) in container: '%s'\n", dest);
                continue;
            }
        } else if ( ( is_dir(source) == 0 ) && ( is_dir(joinpath(container_dir, dest)) < 0 ) ) {
            if ( singularity_registry_get("OVERLAYFS_ENABLED") != NULL ) {
                singularity_priv_escalate();
                singularity_message(VERBOSE3, "Creating bind directory on overlay file system: %s\n", dest);
                if ( s_mkpath(joinpath(container_dir, dest), 0755) < 0 ) {
                    singularity_priv_drop();
                    singularity_message(WARNING, "Could not create bind point directory in container %s: %s\n", dest, strerror(errno));
                    continue;
                }
                singularity_priv_drop();
            } else {
                singularity_message(WARNING, "Non existent bind point (directory) in container: '%s'\n", dest);
                continue;
            }
        }

        singularity_priv_escalate();
        singularity_message(VERBOSE, "Binding '%s' to '%s/%s'\n", source, container_dir, dest);
        if ( singularity_mount(source, joinpath(container_dir, dest), NULL, MS_BIND|MS_NOSUID|MS_NODEV|MS_REC, NULL) < 0 ) {
            singularity_message(ERROR, "There was an error binding the path %s: %s\n", source, strerror(errno));
            ABORT(255);
        }
        if ( singularity_priv_userns_enabled() != 1 ) {
            if ( singularity_mount(NULL, joinpath(container_dir, dest), NULL, MS_BIND|MS_NOSUID|MS_NODEV|MS_REC|MS_REMOUNT, NULL) < 0 ) {
                singularity_message(ERROR, "There was an error remounting the path %s: %s\n", source, strerror(errno));
                ABORT(255);
            }
        }
        singularity_priv_drop();
    }

    return(0);
}

