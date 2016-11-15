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
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/file.h>
#include <sys/mount.h>
#include <unistd.h>
#include <stdlib.h>
#include <grp.h>

#include "util/file.h"
#include "util/util.h"
#include "lib/message.h"
#include "lib/config_parser.h"
#include "lib/image-util.h"
#include "lib/loop-control.h"
#include "lib/privilege.h"

#ifndef LOCALSTATEDIR
#define LOCALSTATEDIR "/var"
#endif


static FILE *image_fp = NULL;
static char *mount_point = NULL;
static char *loop_dev = NULL;
static int read_write = 0;


int rootfs_image_init(char *source, char *mount_dir) {
    singularity_message(DEBUG, "Inializing container rootfs image subsystem\n");
    struct group *file_grp;
    struct group *config_grp;
    struct stat image_stat;
    char *config_gname;

    if ( image_fp != NULL ) {
        singularity_message(WARNING, "Called image_open, but image already open!\n");
        return(1);
    }

    if ( is_file(source) == 0 ) {
        mount_point = strdup(mount_dir);
    } else {
        singularity_message(ERROR, "Container image is not available: %s\n", mount_dir);
        ABORT(255);
    }

    mount_point = strdup(mount_dir);

    if ( envar_defined("SINGULARITY_WRITABLE") == TRUE ) {
        if ( ( image_fp = fopen(source, "r+") ) == NULL ) { // Flawfinder: ignore
            singularity_message(ERROR, "Could not open image (read/write) %s: %s\n", source, strerror(errno));
            ABORT(255);
        }

        if ( envar_defined("SINGULARITY_NOIMAGELOCK") == TRUE ) {
            singularity_message(DEBUG, "Obtaining exclusive write lock on image\n");
            if ( flock(fileno(image_fp), LOCK_EX | LOCK_NB) < 0 ) {
                singularity_message(WARNING, "Could not obtain an exclusive lock on image %s: %s\n", source, strerror(errno));
            }
        }
        read_write = 1;
    } else {
        if ( ( image_fp = fopen(source, "r") ) != NULL ) { // Flawfinder: ignore
            singularity_message(VERBOSE, "Opened image (read only, without privileges) %s\n", source);
        } else {
            singularity_priv_escalate();
            if ( ( image_fp = fopen(source, "r") ) == NULL ) { // Flawfinder: ignore
	            singularity_message(ERROR, "Could not open image (read only, with privileges) %s: %s\n", source, strerror(errno));
                ABORT(255);
            }

            singularity_message(VERBOSE, "Opened image (read only, with privileges) %s\n", source);

            if ( fstat(fileno(image_fp), &image_stat) != 0 ) {
                singularity_message(ERROR, "Could not obtain stat on image %s: %s\n", source, strerror(errno));
                ABORT(255);
            }

            if ( ( file_grp = getgrgid(image_stat.st_gid) ) == NULL ) {
                singularity_message(ERROR, "Could not obtain gid of image %s: %s\n", source, strerror(errno));
                ABORT(255);
            }
            singularity_priv_drop();

            singularity_config_rewind();
            do {
                config_gname = singularity_config_get_value("container group");

                if ( ( has_perm(4, image_stat) == 0 ) || ( has_perm(1, image_stat) == 0 ) ) {
                    singularity_message(VERBOSE, "Image is accessible by calling user\n");
                    break;
                } else {
                    if ( config_gname == NULL ) {
                        singularity_message(ERROR, "Calling user does not have proper permissions to access image, aborting...\n");
                        ABORT(255);
                    }

                    if ( (config_grp = getgrnam(config_gname)) == NULL ) {
                        singularity_message(WARNING, "Unusable container group %s\n", config_gname);
                        continue;
                    }
                    if ( config_grp->gr_gid == file_grp->gr_gid )  {
                        if ( (( image_stat.st_mode & 11 )== 0) && ((image_stat.st_mode & 44 ) == 0) ) {
                            singularity_message(ERROR, "Image does not have proper container group permissions to access image, aborting...\n");
                            ABORT(255);
                        } else {
                            singularity_message(VERBOSE, "Image access is permitted by container group %s specified in config\n", config_gname);
                            break;
                        }
                    }
                }
            } while( config_gname != NULL );


        }
    }

    if ( singularity_image_check(image_fp) < 0 ) {
        singularity_message(ERROR, "File is not a valid Singularity image, aborting...\n");
        ABORT(255);
    }

    if ( ( getuid() != 0 ) && ( is_suid("/proc/self/exe") < 0 ) ) {
        singularity_message(ERROR, "Singularity must be executed in privileged mode to use images\n");
        ABORT(255);
    }

    return(0);
}


int rootfs_image_mount(void) {

    if ( mount_point == NULL ) {
        singularity_message(ERROR, "Called image_mount but image_init() hasn't been called\n");
        ABORT(255);
    }

    if ( image_fp == NULL ) {
        singularity_message(ERROR, "Called image_mount, but image has not been opened!\n");
        ABORT(255);
    }

    if ( is_dir(mount_point) < 0 ) {
        singularity_message(ERROR, "Container directory not available: %s\n", mount_point);
        ABORT(255);
    }

    singularity_message(DEBUG, "Binding image to loop device\n");
    if ( ( loop_dev = singularity_loop_bind(image_fp) ) == NULL ) {
        singularity_message(ERROR, "There was a problem bind mounting the image\n");
        ABORT(255);
    }


    if ( read_write > 0 ) {
        singularity_message(VERBOSE, "Mounting image in read/write\n");
        singularity_priv_escalate();
        if ( mount(loop_dev, mount_point, "ext3", MS_NOSUID, "errors=remount-ro") < 0 ) {
            if ( mount(loop_dev, mount_point, "ext4", MS_NOSUID, "errors=remount-ro") < 0 ) {
                singularity_message(ERROR, "Failed to mount image in (read/write): %s\n", strerror(errno));
                ABORT(255);
            }
        }
        singularity_priv_drop();
    } else {
        singularity_priv_escalate();
        singularity_message(VERBOSE, "Mounting image in read/only\n");
        if ( mount(loop_dev, mount_point, "ext3", MS_NOSUID|MS_RDONLY, "errors=remount-ro") < 0 ) {
            if ( mount(loop_dev, mount_point, "ext4", MS_NOSUID|MS_RDONLY, "errors=remount-ro") < 0 ) {
                singularity_message(ERROR, "Failed to mount image in (read only): %s\n", strerror(errno));
                ABORT(255);
            }
        }
        singularity_priv_drop();
    }


    return(0);
}


