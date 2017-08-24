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
#include <limits.h>
#include <unistd.h>
#include <stdlib.h>
#include <grp.h>
#include <pwd.h>
#include <libgen.h>

#include "util/file.h"
#include "util/util.h"
#include "util/config_parser.h"
#include "util/message.h"
#include "util/privilege.h"
#include "util/suid.h"

#include "../image.h"


int _singularity_image_open(struct image_object *image, int open_flags) {
    struct stat imagestat;

    const char *limit_container_owners = singularity_config_get_value(LIMIT_CONTAINER_OWNERS);
    const char *limit_container_paths = singularity_config_get_value(LIMIT_CONTAINER_PATHS);


    if ( image->fd > 0 ) {
        singularity_message(ERROR, "Called _singularity_image_open() on an open image object: %d\n", image->fd);
        ABORT(255);
    }

    if ( ( is_dir(image->path) == 0 ) && ( open_flags & (O_RDWR|O_WRONLY) ) ) {
        open_flags &= ~(O_RDWR|O_WRONLY) | O_RDONLY;
    }

    singularity_message(DEBUG, "Opening file descriptor to image: %s\n", image->path);
    if ( ( image->fd = open(image->path, open_flags, 0755) ) < 0 ) {
        singularity_message(ERROR, "Could not open image %s: %s\n", image->path, strerror(errno));
        ABORT(255);
    }

    if ( fcntl(image->fd, F_SETFD, FD_CLOEXEC) != 0 ) {
        singularity_message(ERROR, "Could not set file descriptor flag to close on exit: %s\n", strerror(errno));
        ABORT(255);
    }

    if ( fstat(image->fd, &imagestat) < 0 ) {
        singularity_message(ERROR, "Failed calling fstat() on %s (fd: %d): %s\n", image->path, image->fd, strerror(errno));
        ABORT(255);
    }

    image->id = (char *) malloc(intlen((int)imagestat.st_dev) + intlen((long unsigned)imagestat.st_ino) + 2);

    if ( snprintf(image->id, intlen((int)imagestat.st_dev) + intlen((long unsigned)imagestat.st_ino) + 2, "%d.%lu", (int)imagestat.st_dev, (long unsigned)imagestat.st_ino) < 0 ) {
        singularity_message(ERROR, "Failed creating image->id: %s\n", image->path);
        ABORT(255);
    }

    if ( ( singularity_suid_enabled() >= 0 ) && ( singularity_priv_getuid() != 0 ) ) {
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

    return(0);
}

