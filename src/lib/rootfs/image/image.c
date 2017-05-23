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

#include "util/file.h"
#include "util/util.h"
#include "lib/message.h"
#include "lib/config_parser.h"
#include "lib/image/image.h"
#include "lib/loop-control.h"
#include "lib/privilege.h"

#ifndef LOCALSTATEDIR
#define LOCALSTATEDIR "/var"
#endif


static FILE *image_fp = NULL;
static char *mount_point = NULL;
static char *loop_dev = NULL;
static int read_write = 0;

static FILE *open_as_singularity(const char *source, int allow_user_image) {
    FILE *new_fp = NULL;
    singularity_priv_escalate_singularity();
    singularity_message(DEBUG, "Opening image %s as singularity user.\n", source);
    if ( (new_fp = fopen(source, "re")) == NULL ) {
            singularity_message(ERROR, "Could not open image (%s) as 'singularity' user: %s (errno=%d)\n", source, strerror(errno), errno);
            if (allow_user_image) {
               singularity_message(ERROR, "Additionally, could not open image as invoking user.\n");
            } else {
               singularity_message(ERROR, "Additionally, user-owned images are disabled.\n");
            }
            ABORT(255);
    }
    singularity_priv_drop();
    struct stat image_stat;
    if ( fstat(fileno(new_fp), &image_stat) != 0 ) {
        singularity_message(ERROR, "Could not fstat image (%s): %s (errno=%d)\n", source, strerror(errno), errno);
        ABORT(255);
    }
    if ( (image_stat.st_uid != singularity_priv_singularity_uid()) && (image_stat.st_gid != singularity_priv_singularity_gid()) ) {
        singularity_message(ERROR, "In protected mode, the image must be owned by the singularity user or its group (UID=%d or GID=%d).\n", singularity_priv_singularity_uid(), singularity_priv_singularity_gid());
        ABORT(255);
    }
    return new_fp;
}

int rootfs_image_init(char *source, char *mount_dir) {
    singularity_message(DEBUG, "Inializing container rootfs image subsystem\n");

    if ( image_fp != NULL ) {
        singularity_message(WARNING, "Called image_open, but image already open!\n");
        return(1);
    }

    int allow_user_image = singularity_config_get_bool(ALLOW_USER_IMAGE);
    const char *protected_image_mode = singularity_config_get_value(PROTECTED_IMAGE_MODE);
    int protected_image_user = !strcmp(protected_image_mode, "user");
    int protected_image_group = !strcmp(protected_image_mode, "group");
    if (!protected_image_user && !protected_image_group && strcmp(protected_image_mode, "none")) {
        singularity_message(ERROR, "Protected image mode set to %s; known values are 'none', 'user', or 'group'\n", protected_image_mode);
        ABORT(255);
    }
    singularity_message(DEBUG, "Protected image mode set to %s.\n", protected_image_mode);

    if ( is_file(source) == 0 ) {
        mount_point = strdup(mount_dir);
    } else {
        singularity_message(ERROR, "Container image is not available: %s\n", mount_dir);
        ABORT(255);
    }

    mount_point = strdup(mount_dir);

    if ( envar_defined("SINGULARITY_WRITABLE") == TRUE ) {
        if ( !allow_user_image ) {
            singularity_message(ERROR, "Writable image requested, but user images disabled.  Only user images may be writable\n");
            ABORT(255);
        }
        if ( ( image_fp = fopen(source, "r+e") ) == NULL ) { // Flawfinder: ignore
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
    } else if ( allow_user_image && (( image_fp = fopen(source, "re") ) != NULL) ) { // Flawfinder: ignore
        singularity_message(VERBOSE, "Successfully opened image (read only, as invoking user) %s: %s\n", source, strerror(errno));
    } else if (protected_image_user) {
        image_fp = open_as_singularity(source, allow_user_image);
        singularity_message(VERBOSE, "Opened image (read only, as user 'singularity').\n");
    } else if (protected_image_group) {
        image_fp = open_as_singularity(source, allow_user_image);
        // Perform additional group-matching test.
        struct stat image_stat;
        if ( fstat(fileno(image_fp), &image_stat) != 0 ) {
            singularity_message(ERROR, "Could not fstat image (%s): %s (errno=%d)\n", source, strerror(errno), errno);
            ABORT(255);
        }
        if ( !singularity_priv_has_gid(image_stat.st_gid) ) {
            singularity_message(ERROR, "Invoking user is not a member of GID %d, which is required to execute image %s.\n", image_stat.st_gid, source);
            ABORT(255);
        }
    } else {
        singularity_message(ERROR, "Failed to open image %s as invoking user (and privileged mode is disabled): %s (errno=%d).\n", source, strerror(errno), errno);
        ABORT(255);
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
    int opts = MS_NOSUID;

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

    if ( getuid() != 0 ) {
        opts |= MS_NODEV;
    }

    if ( read_write > 0 ) {
        singularity_message(VERBOSE, "Mounting image in read/write\n");
        singularity_priv_escalate();
        if ( mount(loop_dev, mount_point, "ext3", opts, "errors=remount-ro") < 0 ) {
            if ( mount(loop_dev, mount_point, "ext4", opts, "errors=remount-ro") < 0 ) {
                singularity_message(ERROR, "Failed to mount image in (read/write): %s\n", strerror(errno));
                ABORT(255);
            }
        }
        singularity_priv_drop();
    } else {
        singularity_priv_escalate();
        singularity_message(VERBOSE, "Mounting image in read/only\n");
        if ( mount(loop_dev, mount_point, "ext3", opts|MS_RDONLY, "errors=remount-ro") < 0 ) {
            if ( mount(loop_dev, mount_point, "ext4", opts|MS_RDONLY, "errors=remount-ro") < 0 ) {
                singularity_message(ERROR, "Failed to mount image in (read only): %s\n", strerror(errno));
                ABORT(255);
            }
        }
        singularity_priv_drop();
    }


    return(0);
}


