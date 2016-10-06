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

#include "util/file.h"
#include "util/util.h"
#include "lib/message.h"
#include "lib/privilege.h"
#include "lib/config_parser.h"
#include "lib/rootfs/rootfs.h"
#include "lib/ns/ns.h"
#include "../mount-util.h"


void singularity_mount_userbinds(void) {
    char *bind_path_string;
    char *container_dir = singularity_rootfs_dir();

    singularity_message(DEBUG, "Checking for environment variable 'SINGULARITY_BINDPATH'\n");
    if ( ( bind_path_string = envar_path("SINGULARITY_BINDPATH") ) != NULL ) {

        singularity_message(DEBUG, "Checking for 'user bind control' in config\n");
        if ( singularity_config_get_bool("user bind control", 1) <= 0 ) {
            singularity_message(WARNING, "Ignoring user bind request: user bind control is disabled by system administrator\n");
            return;
        }

#ifndef SINGULARITY_NO_NEW_PRIVS
        singularity_message(WARNING, "Ignoring user bind request: host does not support PR_SET_NO_NEW_PRIVS\n");
        return;
#endif

        singularity_message(DEBUG, "Parsing SINGULARITY_BINDPATH for user-specified bind mounts.\n");
        char *outside_token = NULL;
        char *inside_token = NULL;
        char *current = strtok_r(strdup(bind_path_string), ",", &outside_token);

        free(bind_path_string);

        while ( current != NULL ) {
            char *source = strtok_r(current, ":", &inside_token);
            char *dest = strtok_r(NULL, ":", &inside_token);

            current = strtok_r(NULL, ",", &outside_token);

            if ( dest == NULL ) {
                dest = source;
            }

            singularity_message(DEBUG, "Found bind: %s -> container:%s\n", source, dest);

            singularity_message(DEBUG, "Checking if bind point is already mounted: %s\n", dest);
            if ( check_mounted(dest) >= 0 ) {
                singularity_message(WARNING, "Not mounting requested bind point (already mounted in container): %s\n", dest);
                continue;
            }

            if ( ( is_file(source) == 0 ) && ( is_file(joinpath(container_dir, dest)) < 0 ) ) {
                if ( singularity_rootfs_overlay_enabled() > 0 ) {
                    char *dir = dirname(strdup(dest));
                    if ( is_dir(joinpath(container_dir, dir)) < 0 ) {
                        singularity_message(VERBOSE3, "Creating bind directory on overlay file system: %s\n", dest);
                        if ( s_mkpath(joinpath(container_dir, dir), 0755) < 0 ) {
                            singularity_priv_escalate();
                            singularity_message(VERBOSE3, "Retrying with privileges to create bind directory on overlay file system: %s\n", dest);
                            if ( s_mkpath(joinpath(container_dir, dir), 0755) < 0 ) {
                                singularity_message(ERROR, "Could not create basedir for file bind %s: %s\n", dest, strerror(errno));
                                continue;
                            }
                            singularity_priv_drop();
                        }
                    }
                    singularity_priv_escalate();
                    singularity_message(VERBOSE3, "Creating bind file on overlay file system: %s\n", dest);
                    FILE *tmp = fopen(joinpath(container_dir, dest), "w+"); // Flawfinder: ignore
                    singularity_priv_drop();
                    if ( tmp == NULL ) {
                        singularity_message(WARNING, "Skipping user bind, could not create bind point %s: %s\n", dest, strerror(errno));
                        continue;
                    }
                    if ( fclose(tmp) != 0 ) {
                        singularity_message(WARNING, "Skipping user bind, could not close bind point file descriptor %s: %s\n", dest, strerror(errno));
                        continue;
                    }
                    singularity_message(DEBUG, "Created bind file: %s\n", dest);
                } else {
                    singularity_message(WARNING, "Skipping user bind, non existant bind point (file) in container: '%s'\n", dest);
                    continue;
                }
            } else if ( ( is_dir(source) == 0 ) && ( is_dir(joinpath(container_dir, dest)) < 0 ) ) {
                if ( singularity_rootfs_overlay_enabled() > 0 ) {
                    singularity_message(VERBOSE3, "Creating bind directory on overlay file system: %s\n", dest);
                    if ( s_mkpath(joinpath(container_dir, dest), 0755) < 0 ) {
                        singularity_priv_escalate();
                        singularity_message(VERBOSE3, "Retrying with privileges to create bind directory on overlay file system: %s\n", dest);
                        if ( s_mkpath(joinpath(container_dir, dest), 0755) < 0 ) {
                            singularity_priv_drop();
                            singularity_message(WARNING, "Skipping user bind, could not create bind point %s: %s\n", dest, strerror(errno));
                            continue;
                        }
                        singularity_priv_drop();
                    }
                } else {
                    singularity_message(WARNING, "Skipping user bind, non existant bind point (directory) in container: '%s'\n", dest);
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

        singularity_message(DEBUG, "Unsetting environment variable 'SINGULARITY_BINDPATH'\n");
        unsetenv("SINGULARITY_BINDPATH");
    } else {
        singularity_message(DEBUG, "No user bind mounts specified.\n");
    }
}

