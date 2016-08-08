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
#include <unistd.h>
#include <stdlib.h>
#include <libgen.h>

#include "file.h"
#include "util.h"
#include "message.h"
#include "config_parser.h"
#include "privilege.h"
#include "image/image.h"
#include "dir/dir.h"

#define ROOTFS_IMAGE    1
#define ROOTFS_DIR      2
#define ROOTFS_TGZ      3


static int module = 0;
static char *mount_point = NULL;

int singularity_rootfs_init(char *source) {
    char *containername = basename(strdup(source));
    message(DEBUG, "Checking on container source type\n");

    if ( containername != NULL ) {
        setenv("SINGULARITY_CONTAINER", containername, 1);
    } else {
        setenv("SINGULARITY_CONTAINER", "unknown", 1);
    }

    config_rewind();
    message(DEBUG, "Figuring out where to mount Singularity container\n");
    if ( ( mount_point = config_get_key_value("container dir") ) == NULL ) {
        message(DEBUG, "Using default container path of: /var/singularity/mnt\n");
        mount_point = strdup("/var/singularity/mnt");
    }
    message(DEBUG, "Set image mount path to: %s\n", mount_point);

    mount_point = strdup(mount_point);

    if ( is_file(source) == 0 ) {
        module = ROOTFS_IMAGE;
        return(rootfs_image_init(source, mount_point));
    } else if ( is_dir(source) == 0 ) {
        module = ROOTFS_DIR;
        return(rootfs_dir_init(source, mount_point));
    }

    message(ERROR, "Unknown rootfs source type\n");
    ABORT(255);
    return(-1);
}

int singularity_rootfs_mount(void) {
    message(DEBUG, "Mounting image\n");

    if ( module == ROOTFS_IMAGE ) {
        if ( rootfs_image_mount() < 0 ) {
            message(ERROR, "Failed mounting image, aborting...\n");
            ABORT(255);
        }
    } else if ( module == ROOTFS_DIR ) {
        if ( rootfs_dir_mount() < 0 ) {
            message(ERROR, "Failed directory, aborting...\n");
            ABORT(255);
        }
    }

    //TODO: Setup overlay file system here...

    return(0);
}


int singularity_rootfs_chroot(void) {
    message(VERBOSE, "Entering container file system space\n");

    if ( is_exec(joinpath(mount_point, "/bin/sh")) < 0 ) {
        message(ERROR, "Container does not have a valid /bin/sh\n");
        ABORT(255);
    }

    priv_escalate();
    if ( chroot(mount_point) < 0 ) { // Flawfinder: ignore (yep, yep, yep... we know!)
        message(ERROR, "failed enter container at: %s\n", mount_point);
        ABORT(255);
    }
    priv_drop();

    message(DEBUG, "Changing dir to '/' within the new root\n");
    if ( chdir("/") < 0 ) {
        message(ERROR, "Could not chdir after chroot to /: %s\n", strerror(errno));
        ABORT(1);
    }

    return(0);
}
