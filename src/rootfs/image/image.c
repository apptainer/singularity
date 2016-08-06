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

#include "file.h"
#include "util.h"
#include "message.h"
#include "config_parser.h"
#include "image.h"
#include "image-util.h"
#include "loop-control.h"
#include "privilege.h"

#ifndef LOCALSTATEDIR
#define LOCALSTATEDIR "/var"
#endif


static FILE *image_fp = NULL;
static char *mount_point = NULL;
static char *loop_dev = NULL;
static int read_write = 0;


int rootfs_image_init(char *source, char *mount_dir, int writable) {
    message(DEBUG, "Inializing container rootfs image subsystem\n");

    if ( image_fp != NULL ) {
        message(WARNING, "Called image_open, but image already open!\n");
        return(1);
    }

    if ( is_file(source) == 0 ) {
        mount_point = strdup(mount_dir);
    } else {
        message(ERROR, "Container image is not available: %s\n", mount_dir);
        ABORT(255);
    }

    if ( is_dir(mount_dir) == 0 ) {
        mount_point = strdup(mount_dir);
    } else {
        message(ERROR, "Mount point for container image is not a directory: %s\n", mount_dir);
        ABORT(255);
    }

    if ( writable > 0 ) {
        if ( ( image_fp = fopen(source, "r+") ) == NULL ) {
            message(ERROR, "Could not open image (read/write) %s: %s\n", source, strerror(errno));
            ABORT(255);
        }

        message(DEBUG, "Obtaining exclusive write lock on image\n");
        if ( flock(fileno(image_fp), LOCK_EX | LOCK_NB) < 0 ) {
            message(ERROR, "Could not obtain a shared lock on image: %s\n", source);
            ABORT(255);
        }
        read_write = 1;
    } else {
        if ( ( image_fp = fopen(source, "r") ) == NULL ) {
            message(ERROR, "Could not open image (read only) %s: %s\n", source, strerror(errno));
            ABORT(255);
        }
    }

    return(0);
}


int rootfs_image_mount(void) {
    int offset;

    if ( mount_point == NULL ) {
        message(ERROR, "Called image_mount but image_init() hasn't been called\n");
        ABORT(255);
    }

    if ( image_fp == NULL ) {
        message(ERROR, "Called image_mount, but image has not been opened!\n");
        ABORT(255);
    }

    message(DEBUG, "Calculating image offset\n");
// TODO: Move offset to loop functions
    if ( ( offset = image_util_offset(image_fp) ) < 0 ) {
        message(ERROR, "Could not obtain message offset of image\n");
        ABORT(255);
    }

    message(DEBUG, "Binding image to loop device\n");
    if ( ( loop_dev = loop_bind(image_fp, offset) ) == NULL ) {
        message(ERROR, "There was a problem bind mounting the image\n");
        ABORT(255);
    }


    if ( read_write > 0 ) {
        message(VERBOSE, "Mounting image in read/write\n");
        priv_escalate();
        if ( mount(loop_dev, mount_point, "ext3", MS_NOSUID, "errors=remount-ro") < 0 ) {
            message(ERROR, "Failed to mount image!\n");
            ABORT(255);
        }
        priv_drop();
    } else {
        priv_escalate();
        message(VERBOSE, "Mounting image in read/only\n");
        if ( mount(loop_dev, mount_point, "ext3", MS_NOSUID|MS_RDONLY, "errors=remount-ro") < 0 ) {
            message(ERROR, "Failed to mount image!\n");
            ABORT(255);
        }
        priv_drop();
    }


    return(0);
}


int rootfs_image_umount(void) {

    if ( mount_point == NULL ) {
        message(ERROR, "Called image_umount but image_init() hasn't been called\n");
        ABORT(255);
    }

    if ( image_fp == NULL ) {
        message(ERROR, "Called image_umount, but image has not been opened!\n");
        ABORT(255);
    }

    priv_escalate();
    if ( umount(mount_point) < 0 ) {
        message(ERROR, "Failed umounting file system\n");
        ABORT(255);
    }
    priv_drop();

    fclose(image_fp);
    (void) loop_free();

    return(0);
}

