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
#include <sys/stat.h>
#include <sys/types.h>
#include <unistd.h>
#include <fcntl.h>
#include <limits.h>
#include <stdlib.h>
#include <grp.h>
#include <pwd.h>
#include <libgen.h>

#include "util/file.h"
#include "util/util.h"
#include "util/message.h"
#include "util/registry.h"
#include "util/config_parser.h"
#include "util/privilege.h"
#include "util/suid.h"

#include "./image.h"
#include "./bind.h"
#include "./squashfs/include.h"
#include "./dir/include.h"
#include "./ext3/include.h"


struct image_object singularity_image_init(char *path, int open_flags) {
    struct image_object image;
    char *real_path;

    if ( path == NULL ) {
        singularity_message(ERROR, "No container image path defined\n");
        ABORT(255);
    }

    real_path = realpath(path, NULL); // Flawfinder: ignore
    if ( real_path == NULL ) {
        singularity_message(ERROR, "Image path doesn't exists\n");
        ABORT(255);
    }

    image.path = real_path;
    image.name = basename(strdup(real_path));
    image.type = -1;
    image.fd = -1;
    image.loopdev = NULL;
    image.offset = 0;

    if ( open_flags & ( O_RDWR | O_WRONLY ) ) {
        image.writable = 1;
    } else {
        image.writable = 0;
    }

    singularity_message(DEBUG, "Calling image_init for each file system module\n");
    if ( _singularity_image_dir_init(&image, open_flags) == 0 ) {
        singularity_message(DEBUG, "got image_init type for directory\n");
        image.type = DIRECTORY;
        if ( ( singularity_config_get_bool(ALLOW_CONTAINER_DIR) <= 0 ) && ( singularity_priv_getuid() != 0 ) ) {
            singularity_message(ERROR, "Configuration disallows users from running directory based containers\n");
            ABORT(255);
        }
    } else if ( _singularity_image_squashfs_init(&image, open_flags) == 0 ) {
        singularity_message(DEBUG, "got image_init type for squashfs\n");
        image.type = SQUASHFS;
        if ( ( singularity_config_get_bool(ALLOW_CONTAINER_SQUASHFS) <= 0 ) && ( singularity_priv_getuid() != 0 ) ) {
            singularity_message(ERROR, "Configuration disallows users from running squashFS based containers\n");
            ABORT(255);
        }
    } else if ( _singularity_image_ext3_init(&image, open_flags) == 0 ) {
        singularity_message(DEBUG, "got image_init type for ext3\n");
        image.type = EXT3;
        if ( ( singularity_config_get_bool(ALLOW_CONTAINER_EXTFS) <= 0 ) && ( singularity_priv_getuid() != 0 ) ) {
            singularity_message(ERROR, "Configuration disallows users from running extFS based containers\n");
            ABORT(255);
        }
    } else {
        if ( errno == EROFS ) {
            singularity_message(ERROR, "Unable to open squashfs image in read-write mode: %s\n", strerror(errno));
        } else {
            singularity_message(ERROR, "Unknown image format/type: %s\n", path);
        }
        ABORT(255);
    }

    if ( fcntl(image.fd, F_SETFD, FD_CLOEXEC) != 0 ) {
        singularity_message(ERROR, "Failed to set CLOEXEC on image file descriptor\n");
        ABORT(255);
    }

    if ( ( singularity_suid_enabled() >= 0 ) && ( singularity_priv_getuid() != 0 ) ) {
        singularity_limit_container_paths(&image);
        singularity_limit_container_owners(&image);
    }

    return(image);
}

int singularity_image_fd(struct image_object *image) {
    return(image->fd);
}

char *singularity_image_loopdev(struct image_object *image) {
    return(image->loopdev);
}

char *singularity_image_name(struct image_object *image) {
    return(image->name);
}

char *singularity_image_path(struct image_object *image) {
    return(image->path);
}

int singularity_image_offset(struct image_object *image) {
    return(image->offset);
}

int singularity_image_type(struct image_object *image) {
    return(image->type);
}

int singularity_image_writable(struct image_object *image) {
    return(image->writable);
}

int singularity_image_mount(struct image_object *image, char *mount_point) {
    if ( singularity_registry_get("DAEMON_JOIN") ) {
        singularity_message(ERROR, "Internal Error - This function should not be called when joining an instance\n");
    }

    singularity_message(DEBUG, "Figuring out which mount module to use...\n");
    if ( image->type == SQUASHFS ) {
        singularity_message(DEBUG, "Calling squashfs_mount\n");
        return(_singularity_image_squashfs_mount(image, mount_point));
    } else if ( image->type == DIRECTORY ) {
        singularity_message(DEBUG, "Calling dir_mount\n");
        return(_singularity_image_dir_mount(image, mount_point));
    } else if ( image->type == EXT3 ) {
        singularity_message(DEBUG, "Calling ext3_mount\n");
        return(_singularity_image_ext3_mount(image, mount_point));
    } else {
        singularity_message(ERROR, "Can not mount file system of unknown type\n");
        ABORT(255);
    }
    return(-1);
}

void singularity_limit_container_owners(struct image_object *image) {
    const char *limit_container_owners = singularity_config_get_value(LIMIT_CONTAINER_OWNERS);

    if ( strcmp(limit_container_owners, "NULL") != 0 ) {
        struct stat image_stat;
        char *user_token = NULL;
        char *current = strtok_r(strdup(limit_container_owners), ",", &user_token);

        chomp(current);

        singularity_message(DEBUG, "Limiting container access to allowed users\n");

        if ( fstat(image->fd, &image_stat) != 0 ) {
            singularity_message(ERROR, "Could not fstat() image file descriptor (%d): %s\n", image->fd, strerror(errno));
            ABORT(255);
        }

        while (1) {
            struct passwd *user_pw;

            if ( current[0] == '\0' ) {
                singularity_message(DEBUG, "Skipping blank user limit entry\n");
            } else {
                singularity_message(DEBUG, "Checking user: '%s'\n", current);

                if ( ( user_pw = getpwnam(current) ) != NULL ) {
                    if ( user_pw->pw_uid == image_stat.st_uid ) {
                        singularity_message(DEBUG, "Singularity image is owned by required user: %s\n", current);
                        break;
                    }
                }
            }

            current = strtok_r(NULL, ",", &user_token);
            chomp(current);

            if ( current == NULL ) {
                singularity_message(ERROR, "Singularity image is not owned by required user(s)\n");
                ABORT(255);
            }
        }
    }
}

void singularity_limit_container_paths(struct image_object *image) {
    const char *limit_container_paths = singularity_config_get_value(LIMIT_CONTAINER_PATHS);

    if ( strcmp(limit_container_paths, "NULL") != 0 ) { 
        char image_path[PATH_MAX];
        char *path_token = NULL;
        char *fd_path = NULL;

        fd_path = (char *) malloc(PATH_MAX+21);

        singularity_message(DEBUG, "Obtaining full path to image file descriptor (%d)\n", image->fd);

        if ( snprintf(fd_path, PATH_MAX+20, "/proc/self/fd/%d", image->fd) > 0 ) {
            singularity_message(DEBUG, "Checking image path from file descriptor source: %s\n", fd_path);
        } else {
            singularity_message(ERROR, "Internal - Failed allocating memory for fd_path string: %s\n", strerror(errno));
            ABORT(255);
        }

        if ( readlink(fd_path, image_path, PATH_MAX-1) > 0 ) { // Flawfinder: ignore (TOCTOU not an issue within /proc)
            char *current = strtok_r(strdup(limit_container_paths), ",", &path_token);

            chomp(current);
            while (1) {

                if ( current[0] == '\0' ) {
                    singularity_message(DEBUG, "Skipping blank path limit entry\n");

                } else {
                    singularity_message(DEBUG, "Checking image path: '%s'\n", current);

                    if ( strncmp(image_path, current, strlength(current, PATH_MAX)) == 0 ) {
                        singularity_message(VERBOSE, "Singularity image is in an allowed path: %s\n", current);
                        break;
                    }

                    current = strtok_r(NULL, ",", &path_token);
                    chomp(current);

                    if ( current == NULL ) {
                        singularity_message(ERROR, "Singularity image is not in an allowed configured path\n");
                        ABORT(255);
                    }
                }
            }

        } else {
            singularity_message(ERROR, "Could not obtain the full system path of the image file: %s\n", strerror(errno));
            ABORT(255);
        }
    }
}
