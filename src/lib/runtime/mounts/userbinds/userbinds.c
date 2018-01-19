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


int _singularity_runtime_mount_userbinds(void) {
    char *container_dir = CONTAINER_FINALDIR;
    char *bind_path_string;

    singularity_message(DEBUG, "Checking for environment variable 'SINGULARITY_BINDPATH'\n");
    if ( ( bind_path_string = singularity_registry_get("BINDPATH") ) != NULL ) {

        singularity_message(DEBUG, "Checking for 'user bind control' in config\n");
        if ( singularity_config_get_bool(USER_BIND_CONTROL) <= 0 ) {
            singularity_message(WARNING, "Ignoring user bind request: user bind control is disabled by system administrator\n");
            return(0);
        }

#ifndef SINGULARITY_NO_NEW_PRIVS
        singularity_message(WARNING, "Ignoring user bind request: host does not support PR_SET_NO_NEW_PRIVS\n");
        return(0);
#endif

        singularity_message(DEBUG, "Parsing SINGULARITY_BINDPATH for user-specified bind mounts.\n");
        char *outside_token = NULL;
        char *inside_token = NULL;
        char *current = strtok_r(strdup(bind_path_string), ",", &outside_token);

        free(bind_path_string);

        while ( current != NULL ) {
            int read_only = 0;
            char *source = strtok_r(current, ":", &inside_token);
            char *dest = strtok_r(NULL, ":", &inside_token);
            char *opts = strtok_r(NULL, ":", &inside_token);

            current = strtok_r(NULL, ",", &outside_token);

            if ( dest == NULL ) {
                dest = source;
            }

            singularity_message(DEBUG, "Found bind: %s -> container:%s\n", source, dest);

            if ( opts != NULL ) {
                if ( strcmp(opts, "rw") == 0 ) {
                    // This is the default
                } else if ( strcmp(opts, "ro") == 0 ) {
                    read_only = 1;
                } else {
                    singularity_message(WARNING, "Not mounting requested bind point, invalid mount option %s: %s\n", opts, dest);
                    continue;
                }
            }


            singularity_message(DEBUG, "Checking if bind point is already mounted: %s\n", dest);
            if ( check_mounted(dest) >= 0 ) {
                singularity_message(WARNING, "Not mounting requested bind point (already mounted in container): %s\n", dest);
                continue;
            }

            if ( ( is_file(source) == 0 ) && ( is_file(joinpath(container_dir, dest)) < 0 ) ) {
                if ( singularity_registry_get("OVERLAYFS_ENABLED") != NULL ) {
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
                    singularity_message(WARNING, "Skipping user bind, non existent bind point (file) in container: '%s'\n", dest);
                    continue;
                }
            } else if ( ( is_dir(source) == 0 ) && ( is_dir(joinpath(container_dir, dest)) < 0 ) ) {
                if ( singularity_registry_get("OVERLAYFS_ENABLED") != NULL ) {
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
                    singularity_message(WARNING, "Skipping user bind, non existent bind point (directory) in container: '%s'\n", dest);
                    continue;
                }
            }

            singularity_priv_escalate();
            singularity_message(VERBOSE, "Binding '%s' to '%s/%s'\n", source, container_dir, dest);
            if ( singularity_mount(source, joinpath(container_dir, dest), NULL, MS_BIND|MS_NOSUID|MS_NODEV|MS_REC, NULL) < 0 ) {
                singularity_message(ERROR, "There was an error binding the path %s: %s\n", source, strerror(errno));
                ABORT(255);
            }
            if ( read_only ) {
                if ( singularity_priv_userns_enabled() == 1 ) {
                    singularity_message(WARNING, "Can not make bind mount read only within the user namespace: %s\n", dest);
                } else {
                    singularity_message(VERBOSE, "Remounting %s read-only\n", dest);
                    if ( singularity_mount(NULL, joinpath(container_dir, dest), NULL, MS_RDONLY|MS_BIND|MS_NOSUID|MS_NODEV|MS_REC|MS_REMOUNT, NULL) < 0 ) {
                        singularity_message(ERROR, "There was an error write-protecting the path %s: %s\n", source, strerror(errno));
                        ABORT(255);
                    }
                    if ( access(joinpath(container_dir, dest), W_OK) == 0 || (errno != EROFS && errno != EACCES) ) { // Flawfinder: ignore (precautionary confirmation, not necessary)
                        singularity_message(ERROR, "Failed to write-protect the path %s: %s\n", source, strerror(errno));
                        ABORT(255);
                    }
                }
            } else {
                if ( singularity_priv_userns_enabled() <= 0 ) {
                    if ( singularity_mount(NULL, joinpath(container_dir, dest), NULL, MS_BIND|MS_NOSUID|MS_NODEV|MS_REC|MS_REMOUNT, NULL) < 0 ) {
                        singularity_message(ERROR, "There was an error remounting the path %s: %s\n", source, strerror(errno));
                        ABORT(255);
                    }
                }
            }
            singularity_priv_drop();

        }

        singularity_message(DEBUG, "Unsetting environment variable 'SINGULARITY_BINDPATH'\n");
        unsetenv("SINGULARITY_BINDPATH");
    } else {
        singularity_message(DEBUG, "No user bind mounts specified.\n");
    }
    return(0);
}

