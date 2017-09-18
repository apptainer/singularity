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

#include <stdlib.h>
#include <linux/loop.h>
#include <unistd.h>
#include <sys/file.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/sysmacros.h>
#include <errno.h> 
#include <string.h>
#include <fcntl.h>
#include <sys/ioctl.h>

#include "config.h"
#include "lib/image/image.h"
#include "util/util.h"
#include "util/file.h"
#include "util/message.h"
#include "util/privilege.h"
#include "util/registry.h"
#include "util/config_parser.h"

#include "./image.h"

#ifndef LO_FLAGS_AUTOCLEAR
#define LO_FLAGS_AUTOCLEAR 4
#endif


char *singularity_image_bind(struct image_object *image) {
    struct loop_info64 lo64 = {0};
    long int max_loop_devs;
    const char *max_loop_devs_string = singularity_config_get_value(MAX_LOOP_DEVS);
    char *loop_dev = NULL;
    int loop_fd = -1;
    int loop_mount_opts;
    int i;

    singularity_message(DEBUG, "Entered singularity_image_bind()\n");

    singularity_message(DEBUG, "Converting max_loop_devs_string to int: '%s'\n", max_loop_devs_string);
    if ( str2int(max_loop_devs_string, &max_loop_devs) != 0 ) {
        singularity_message(ERROR, "Failed converting config option '%s = %s' to integer\n", MAX_LOOP_DEVS, max_loop_devs_string);
        ABORT(255);
    }
    singularity_message(DEBUG, "Converted max_loop_devs_string to int: '%s' -> %ld\n", max_loop_devs_string, max_loop_devs);

    singularity_message(DEBUG, "Checking if this image has been properly opened\n");
    if ( image->fd <= 0 ) {
        singularity_message(ERROR, "Image file descriptor not assigned!\n");
        ABORT(255);
    }

    if ( image->writable <= 0 ) {
        loop_mount_opts = O_RDONLY;
    } else {
        loop_mount_opts = O_RDWR;
    }

    singularity_priv_escalate();
    singularity_message(DEBUG, "Finding next available loop device...\n");
    for( i=0; i < max_loop_devs; i++ ) {
        char *test_loopdev = strjoin("/dev/loop", int2str(i));

        if ( is_blk(test_loopdev) < 0 ) {
            singularity_message(DEBUG, "Instantiating loop device: %s\n", test_loopdev);
            if ( mknod(test_loopdev, S_IFBLK | 0644, makedev(7, i)) < 0 ) {
                if ( errno != EEXIST ) {
                    singularity_message(ERROR, "Could not create %s: %s\n", test_loopdev, strerror(errno));
                    ABORT(255);
                }
            }
        }

        if ( ( loop_fd = open(test_loopdev, loop_mount_opts) ) < 0 ) { // Flawfinder: ignore
            singularity_message(VERBOSE, "Could not open loop device %s: %s\n", test_loopdev, strerror(errno));
            continue;
        }

        if ( ioctl(loop_fd, LOOP_SET_FD, image->fd)== 0 ) {
            loop_dev = strdup(test_loopdev);
            break;
        } else {
            if ( errno == EBUSY ) {
                close(loop_fd);
                continue;
            } else {
                singularity_message(WARNING, "Could not associate image to loop %s: %s\n", test_loopdev, strerror(errno));
                close(loop_fd);
                continue;
            }
        }

    }
    singularity_priv_drop();

    if ( loop_dev == NULL ) {
        singularity_message(ERROR, "No more available loop devices, try increasing '%s' in singularity.conf\n", MAX_LOOP_DEVS);
        ABORT(255);
    }

    singularity_message(VERBOSE, "Found available loop device: %s\n", loop_dev);

    singularity_message(DEBUG, "Setting LO_FLAGS_AUTOCLEAR\n");
    lo64.lo_flags = LO_FLAGS_AUTOCLEAR;

    singularity_message(DEBUG, "Using image offset: %d\n", image->offset);
    lo64.lo_offset = image->offset;

    singularity_priv_escalate();
    singularity_message(DEBUG, "Setting loop device flags\n");
    if ( ioctl(loop_fd, LOOP_SET_STATUS64, &lo64) < 0 ) {
        singularity_message(ERROR, "Failed to set loop flags on loop device: %s\n", strerror(errno));
        (void)ioctl(loop_fd, LOOP_CLR_FD, 0);
        ABORT(255);
    }
    singularity_priv_drop();

    singularity_message(VERBOSE, "Using loop device: %s\n", loop_dev);

    if ( fcntl(loop_fd, F_SETFD, FD_CLOEXEC) != 0 ) {
        singularity_message(ERROR, "Could not set file descriptor flag to close on exit: %s\n", strerror(errno));
        ABORT(255);
    }

    return(loop_dev);
}

