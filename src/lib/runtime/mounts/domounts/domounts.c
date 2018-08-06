/* 
 * Copyright (c) 2017-2018, SyLabs, Inc. All rights reserved.
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
#include "util/registry.h"
#include "util/mount.h"
#include "util/mountlist.h"


int _singularity_runtime_domounts(struct mountlist *mountlist) {
    char *container_dir = CONTAINER_FINALDIR;
    char *source = NULL;
    char *target = NULL;
    struct mountlist_point *point;

    for (point = mountlist->first; point != NULL; point = point->next) {
        source = (char *) point->source;
        if (source == NULL)
            source = (char *) point->target;
        if (target != NULL)
            free(target);
        target = joinpath(container_dir, point->target);

        if ( check_mounted(point->target) >= 0 ) {
            singularity_message(WARNING, "Not mounting %s (already mounted in container)\n", point->target);
            continue;
        }

        if ( ( is_file(source) == 0 ) && ( is_file(target) < 0 ) ) {
            if ( point->mountlistflags & ML_ONLY_IF_POINT_PRESENT ) {
                singularity_message(VERBOSE, "Not mounting '%s', file does not exist within container\n", source);
                continue;
            }
            if ( singularity_registry_get("OVERLAYFS_ENABLED") != NULL ) {
                char *basedir = strdup(target);
                basedir = dirname(basedir);

                singularity_message(DEBUG, "Checking base directory for file %s ('%s')\n", target, basedir);
                if ( is_dir(basedir) != 0 ) {
                    singularity_message(DEBUG, "Creating base directory for file mount\n");
                    if ( container_mkpath_priv(basedir, 0755) != 0 ) {
                        singularity_message(ERROR, "Failed creating base directory for mounted file: %s\n", target);
                        ABORT(255);
                    }
                }

                free(basedir);

                singularity_message(VERBOSE3, "Creating file mountpoint on overlay file system: %s\n", target);
                if ( fileput_priv(target, "") != 0 ) {
                    continue;
                }
                singularity_message(DEBUG, "Created bind file: %s\n", target);
            } else {
                singularity_message(WARNING, "Non existent mount point (file) in container: '%s'\n", target);
                continue;
            }
        } else if ( ( is_dir(source) == 0 ) && ( is_dir(target) < 0 ) ) {
            if ( point->mountlistflags & ML_ONLY_IF_POINT_PRESENT ) {
                singularity_message(VERBOSE, "Not mounting '%s', directory does not exist within container\n", source);
                continue;
            }
            if ( singularity_registry_get("OVERLAYFS_ENABLED") != NULL ) {
                singularity_message(VERBOSE3, "Creating mount directory on overlay file system: %s\n", target);
                if ( container_mkpath_priv(target, 0755) < 0 ) {
                    singularity_message(WARNING, "Could not create mount point directory in container %s: %s\n", target, strerror(errno));
                    continue;
                }
            } else {
                singularity_message(WARNING, "Non existent mountpoint (directory) in container: '%s'\n", target);
                continue;
            }
        }

        singularity_message(VERBOSE, "Mounting '%s' at '%s'\n", source, target);
        unsigned long read_only = point->mountflags & MS_RDONLY;
        point->mountflags &= ~MS_RDONLY;
        if ( singularity_mount_point(point) < 0 ) {
            singularity_message(ERROR, "There was an error mounting %s: %s\n", source, strerror(errno));
            ABORT(255);
        }

        if ( read_only ) {
            if ( singularity_priv_userns_enabled() == 1 ) {
                singularity_message(WARNING, "Can not make mount read only within the user namespace: %s\n", target);
            } else {
                singularity_message(VERBOSE, "Remounting %s read-only\n", target);
                point->mountflags |= MS_REMOUNT|MS_RDONLY;
                if ( singularity_mount_point(point) < 0 ) {
                    singularity_message(ERROR, "There was an error write-protecting the path %s: %s\n", source, strerror(errno));
                    ABORT(255);
                }
                if ( access(target, W_OK) == 0 || (errno != EROFS && errno != EACCES) ) { // Flawfinder: ignore (precautionary confirmation, not necessary)
                    singularity_message(ERROR, "Failed to write-protect the path %s: %s\n", source, strerror(errno));
                    ABORT(255);
                }
            }
        } else if ( singularity_priv_userns_enabled() != 1 ) {
            point->mountflags |= MS_REMOUNT;
            singularity_message(VERBOSE, "Remounting %s\n", target);
            if ( singularity_mount_point(point) < 0 ) {
                singularity_message(ERROR, "There was an error remounting the path %s: %s\n", source, strerror(errno));
                ABORT(255);
            }
        }
    }

    if (target != NULL)
        free(target);
    
    return(0);
}
