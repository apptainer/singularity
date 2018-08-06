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
#include <dirent.h>
#include <sys/mount.h>
#include <sys/stat.h>
#include <unistd.h>
#include <stdlib.h>
#include <libgen.h>

#include "config.h"
#include "util/config_parser.h"
#include "util/file.h"
#include "util/util.h"
#include "util/message.h"
#include "util/privilege.h"
#include "util/registry.h"
#include "util/mount.h"
#include "util/mountlist.h"

void singularity_runtime_domounts_init(struct mountlist *mountlist) {
    memset(mountlist, 0, sizeof(*mountlist));

    singularity_registry_set("UNDERLAY_ENABLED", NULL);
    if ( ( singularity_config_get_bool_char(ENABLE_UNDERLAY) > 0 ) ) {
        if ( singularity_registry_get("DISABLE_UNDERLAY") != NULL ) {
            singularity_message(VERBOSE3, "Not enabling underlay via environment\n");
        } else {
            singularity_message(VERBOSE3, "Enabling underlay\n");
            singularity_registry_set("UNDERLAY_ENABLED", "1");
        }
    }
}

static void bind_image_final(char *source, char *sub_path) {
    char *target = joinpath(CONTAINER_FINALDIR, sub_path);
    singularity_message(VERBOSE3, "Binding %s to %s\n", source, target);
    if ( singularity_mount(source, target, NULL, MS_BIND|MS_NOSUID|MS_REC, NULL) < 0 ) {

        singularity_message(ERROR, "Failed binding %s to %s\n", source, target);
        ABORT(255);
    }
    free(target);
}

static void mount_missing(char *image_path, char *underlay_path, char *sub_path) {
    DIR *dir;
    struct dirent *dp; 
    char *source = NULL;
    char *target = NULL;
    char *existing_path = NULL;
    int binds = 0;

    singularity_message(VERBOSE3, "Mounting missing files/directories from %s\n", image_path);

    // First find an existing mountpoint inside the underlay directory, if any
    if ( ( dir = opendir(underlay_path) ) == NULL ) {
        singularity_message(ERROR, "Could not open underlay dir %s", underlay_path);
        ABORT(255);
    }
    while ( (dp = readdir(dir) ) != NULL ) {
        if ( ( strcmp(dp->d_name, ".") != 0 ) &&
             ( strcmp(dp->d_name, "..") != 0 ) ) {
            break;
        }
    }
    if ( dp == NULL ) {
        singularity_message(VERBOSE3, "Skipping empty underlay directory: %s\n", underlay_path);
        closedir(dir);
        return;
    }
    singularity_message(DEBUG, "There is at least one mountpoint in %s: %s\n", underlay_path, dp->d_name);
    existing_path = joinpath(sub_path, dp->d_name);
    closedir(dir);

    // Now search through the image for missing mountpoints in the underlay
    if ( ( dir = opendir(image_path) ) == NULL ) {
        singularity_message(ERROR, "Could not open dir %s", image_path);
        ABORT(255);
    }

    while ( ( dp = readdir(dir) ) != NULL ) {
        if ( (strcmp(dp->d_name, ".") == 0 ) || (strcmp(dp->d_name, "..") == 0 ) )
            continue;
        if ( source != NULL )
            free(source);
        if ( target != NULL )
            free(target);
        source = joinpath(image_path, dp->d_name);
        target = joinpath(underlay_path, dp->d_name);
        char *new_sub_path;
        if ( strcmp(sub_path, "/") == 0 )
            new_sub_path = strdup(dp->d_name);
        else
            new_sub_path = joinpath(sub_path, dp->d_name);
        struct stat statbuf;
        int statret = lstat(target, &statbuf);

        if ( is_link(source) == 0 ) {
            if ( statret < 0 ) {
                char link[PATH_MAX+1];
                ssize_t linksize = readlink(source, link, PATH_MAX); // Flawfinder: ignore not controllable by user
                if ( linksize <= 0 ) {
                    singularity_message(WARNING, "Failure reading link info from %s, skipping: %s\n", source, strerror(errno));
                } else { 
                    link[linksize] = '\0';
                    singularity_message(VERBOSE3, "Creating symlink on underlay file system: %s->%s\n", target, link);
                    singularity_priv_escalate();
                    if ( symlink(link, target) < 0 )
                        singularity_message(WARNING, "Failure making link to %s at %s, skipping: %s\n", target, source, strerror(errno));
                    singularity_priv_drop();
                }
            } else if ( S_ISDIR(statbuf.st_mode) ) {
                // It has been replaced by a directory, recurse into it
                mount_missing(source, target, new_sub_path);
            } else {
                singularity_message(VERBOSE3, "Link point on underlay file system already exists, skipping: %s\n", target);
            }
        } else if ( is_file(source) == 0 ) {
            if ( statret < 0 ) {
                singularity_message(VERBOSE3, "Creating file mountpoint on underlay file system: %s\n", target);
                if ( fileput_priv(target, "") != 0 ) {
                    singularity_message(ERROR, "Failed creating underlay file mountpoint: %s\n", target);
                    ABORT(255);
                }
                bind_image_final(source, new_sub_path);
                binds++;
            } else {
                singularity_message(VERBOSE3, "File mountpoint on underlay file system already exists, skipping: %s\n", target);
            }
        } else if ( is_dir(source) == 0 ) {
            if ( statret < 0 ) {
                singularity_message(VERBOSE3, "Creating directory mountpoint on underlay file system: %s\n", target);
                if ( container_mkpath_priv(target, 0755) < 0 ) {
                    singularity_message(ERROR, "Failed creating underlay directory mountpoint: %s\n", target);
                    ABORT(255);
                }
                bind_image_final(source, new_sub_path);
                binds++;
            } else if ( S_ISDIR(statbuf.st_mode) ) {
                mount_missing(source, target, new_sub_path);
            } else {
                singularity_message(VERBOSE3, "Skipping non-directory target with directory source: %s\n", target);
            }
        } else {
            singularity_message(VERBOSE3, "Skipping source that is neither file nor directory nor symlink: %s\n", source);
        }
        free(new_sub_path);
    }

    if ( source != NULL )
        free(source);
    if ( target != NULL )
        free(target);
    closedir(dir);

    if ( binds > 50 ) {
        singularity_message(WARNING, "Underlay of /%s required more than 50 (%d) bind mounts\n",
                existing_path, binds);
    } else {
        singularity_message(DEBUG, "Did %d bind mounts around /%s\n",
                binds, existing_path);
    }
    free(existing_path);
}

static int do_mounts(struct mountlist *mountlist, int overlay) {
    char *container_dir = CONTAINER_FINALDIR;
    char *source = NULL;
    char *target = NULL;
    struct mountlist_point *point;

    for (point = mountlist->first; point != NULL; point = point->next) {
        source = (char *) point->source;
        if ( source == NULL )
            source = (char *) point->target;
        if ( target != NULL )
            free(target);
        target = joinpath(container_dir, point->target);

        if ( check_mounted(point->target) >= 0 ) {
            // make the message only information if ML_ONLY_IF_POINT_PRESENT
            int msglevel = ( point->mountlistflags & ML_ONLY_IF_POINT_PRESENT ) ? VERBOSE : WARNING;
            singularity_message(msglevel, "Not mounting %s (already mounted in container)\n", point->target);
            continue;
        }

        if ( ( is_file(source) == 0 ) && ( is_file(target) < 0 ) ) {
            if ( point->mountlistflags & ML_ONLY_IF_POINT_PRESENT ) {
                singularity_message(VERBOSE, "Not mounting '%s', file does not exist within container\n", source);
                continue;
            }
            if ( overlay ) {
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
            if ( overlay ) {
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
        int read_only = ( (point->mountflags & MS_RDONLY) != 0 );
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

    if ( target != NULL )
        free(target);
    
    return(0);
}

static int underlay_mounts(struct mountlist *mountlist) {
    char *underlay_dir = joinpath(singularity_registry_get("SESSIONDIR"), "underlay");
    char *image_dir = CONTAINER_MOUNTDIR;
    char *final_dir = CONTAINER_FINALDIR;
    char *source = NULL;
    char *underlay_target = NULL;
    char *image_target = NULL;
    struct mountlist_point *point;

    singularity_message(DEBUG, "Creating directory for underlay: %s\n", underlay_dir);
    if ( container_mkpath_priv(underlay_dir, 0755) < 0 ) {
        singularity_message(ERROR, "Failed creating underlay directory %s: %s\n", underlay_dir, strerror(errno));
        ABORT(255);
    }

    singularity_message(DEBUG, "Unmounting final dir %s\n", final_dir);
    singularity_priv_escalate();
    if ( umount(final_dir) != 0 ) {
        singularity_message(ERROR, "Could not umount final directory %s: %s\n", final_dir, strerror(errno));
        ABORT(255);
    }
    singularity_priv_drop();

    singularity_message(DEBUG, "Binding underlay directory to final directory %s->%s\n", underlay_dir, final_dir);
    if ( singularity_mount(underlay_dir, final_dir, NULL, MS_BIND|MS_NOSUID|MS_REC, NULL) < 0 ) {
        singularity_message(ERROR, "Could not bind mount underlay directory to final directory %s->%s: %s\n", underlay_dir, final_dir, strerror(errno));
        ABORT(255);
    }

    // It's important for the underlay directory to be mounted read-only
    //   because otherwise when running unprivileged the code inside the
    //   container would be able to modify it's own root filesystem.
    //   It's not necessary when running setuid because then the
    //   root filesystem directory is owned by root, but do the same
    //   thing anyway for consistency.
    singularity_message(DEBUG, "Remounting underlay directory to final directory read-only %s->%s\n", underlay_dir, final_dir);
    if ( singularity_mount(underlay_dir, final_dir, NULL, MS_REMOUNT|MS_BIND|MS_NOSUID|MS_REC|MS_RDONLY, NULL) < 0 ) {
        singularity_message(ERROR, "Could not re-mount underlay directory to final directory read-only %s->%s: %s\n", underlay_dir, final_dir, strerror(errno));
        ABORT(255);
    }
    errno = 0;
    if ( access(final_dir, W_OK) == 0 || (errno != EROFS && errno != EACCES) ) { // Flawfinder: ignore (precautionary confirmation, not necessary)
        singularity_message(ERROR, "Failed to write-protect the final directory %s: %s\n", final_dir, strerror(errno));
        ABORT(255);
    }

    // make missing mount points in the underlay area
    for (point = mountlist->first; point != NULL; point = point->next) {
        if ( point->mountlistflags & ML_ONLY_IF_POINT_PRESENT )
            continue;

        source = (char *) point->source;
        if ( source == NULL )
            source = (char *) point->target;
        if ( underlay_target != NULL )
            free(underlay_target);
        if ( image_target != NULL )
            free(image_target);
        underlay_target = joinpath(underlay_dir, point->target);
        image_target = joinpath(image_dir, point->target);
        char *basedir = strdup(underlay_target);
        basedir = dirname(basedir);

        if ( ( is_file(source) == 0 ) && ( is_file(underlay_target) < 0 ) &&
               ( ( is_file(image_target) < 0 ) || ( is_dir(basedir) == 0 ) ) ) {

            singularity_message(DEBUG, "Checking base directory for file %s ('%s')\n", underlay_target, basedir);
            if ( is_dir(basedir) != 0 ) {
                singularity_message(DEBUG, "Creating base directory for file mount\n");
                if ( container_mkpath_priv(basedir, 0755) != 0 ) {
                    singularity_message(ERROR, "Failed creating base directory for mounted file: %s\n", underlay_target);
                    ABORT(255);
                }
            }


            singularity_message(VERBOSE3, "Creating file mountpoint on underlay file system: %s\n", underlay_target);
            if ( fileput_priv(underlay_target, "") != 0 ) {
                singularity_message(ERROR, "Could not create mount point file in underlay %s: %s\n", underlay_target, strerror(errno));
                ABORT(255);
            }
            singularity_message(DEBUG, "Created bind file: %s\n", underlay_target);
        } else if ( ( ( point->filesystemtype != NULL ) ||
                      ( is_dir(source) == 0 ) ) &&
                  ( is_dir(underlay_target) < 0 ) &&
                  ( ( is_dir(image_target) < 0 ) ||
                    ( is_dir(basedir) == 0 ) ) ) {
            singularity_message(VERBOSE3, "Creating mount directory on underlay file system: %s\n", underlay_target);
            if ( container_mkpath_priv(underlay_target, 0755) < 0 ) {
                singularity_message(ERROR, "Could not create mount point directory in underlay %s: %s\n", underlay_target, strerror(errno));
                ABORT(255);
            }
        }
        free(basedir);
    }

    if ( image_target != NULL )
        free(image_target);
    if ( underlay_target != NULL )
        free(underlay_target);
    
    // mount everything else from the image into the underlay area
    mount_missing(image_dir, underlay_dir, "/");

    free(underlay_dir);

    // finally, do the requested mounts
    return(do_mounts(mountlist, 0));
}

int _singularity_runtime_domounts(struct mountlist *mountlist) {
    if ( singularity_registry_get("OVERLAYFS_ENABLED") != NULL )
        return(do_mounts(mountlist, 1));

    if ( singularity_registry_get("UNDERLAY_ENABLED") != NULL )
        return(underlay_mounts(mountlist));

    return(do_mounts(mountlist, 0));
}
