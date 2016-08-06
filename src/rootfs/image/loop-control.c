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


#include <stdio.h>
#include <stdlib.h>
#include <linux/loop.h>
#include <unistd.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <errno.h> 
#include <string.h>
#include <fcntl.h>
#include <sys/ioctl.h>

#include "config.h"
#include "loop-control.h"
#include "util.h"
#include "file.h"
#include "image.h"
#include "message.h"
#include "privilege.h"

#ifndef LO_FLAGS_AUTOCLEAR
#define LO_FLAGS_AUTOCLEAR 4
#endif

#define MAX_LOOP_DEVS 128

char *loop_dev;
FILE *loop_fp;

char *loop_bind(FILE *image_fp, int offset) {
    struct loop_info64 lo64 = {0};
    int i;

    message(DEBUG, "Called loop_bind(image_fp, **{loop_dev)\n");

    priv_escalate();

#ifdef LO_FLAGS_AUTOCLEAR
    lo64.lo_flags = LO_FLAGS_AUTOCLEAR;
#endif
    lo64.lo_offset = offset;

    message(DEBUG, "Finding next available loop device...\n");
    for( i=0; i < MAX_LOOP_DEVS; i++ ) {
        char *test_loopdev = strjoin("/dev/loop", int2str(i));

        if ( is_blk(test_loopdev) < 0 ) {
            if ( mknod(test_loopdev, S_IFBLK | 0644, makedev(7, i)) < 0 ) {
                message(ERROR, "Could not create %s: %s\n", test_loopdev, strerror(errno));
                ABORT(255);
            }
        }

        if ( ( loop_fp = fopen(test_loopdev, "r+") ) == NULL ) { // Flawfinder: ignore (not user modifyable)
            message(VERBOSE, "Could not open loop device %s: %s\n", test_loopdev, strerror(errno));
            continue;
        }

        if ( ioctl(fileno(loop_fp), LOOP_SET_FD, fileno(image_fp))== 0 ) {
            loop_dev = strdup(test_loopdev);
            break;
        } else {
            if ( errno == 16 ) {
                fclose(loop_fp);
                continue;
            } else {
                message(WARNING, "Could not associate image to loop %s: %s\n", test_loopdev, strerror(errno));
                fclose(loop_fp);
                continue;
            }
        }

    }

    message(VERBOSE, "Found avaialble loop device: %s\n", loop_dev);

    message(DEBUG, "Setting loop device flags\n");
    if ( ioctl(fileno(loop_fp), LOOP_SET_STATUS64, &lo64) < 0 ) {
        fprintf(stderr, "ERROR: Failed to set loop flags on loop device: %s\n", strerror(errno));
        (void)ioctl(fileno(loop_fp), LOOP_CLR_FD, 0);
        (void)loop_free(loop_dev);
        ABORT(255);
    }

    priv_drop();

    message(VERBOSE, "Using loop device: %s\n", loop_dev);

    message(DEBUG, "Returning loop_bind(image_fp) = loop_fp\n");

    return(loop_dev);

    message(ERROR, "No valid loop devices available\n");
    ABORT(255);

    return(NULL);
}


int loop_free(void) {

    message(DEBUG, "Called loop_free(%s)\n", loop_dev);

    if ( is_blk(loop_dev) < 0 ) {
        message(ERROR, "Loop device is not a valid block device: %s\n", loop_dev);
        ABORT(255);
    }

    if ( ( loop_fp = fopen(loop_dev, "r") ) == NULL ) { // Flawfinder: ignore (only opening read only, and must be a block device)
        message(VERBOSE, "Could not open loop device %s: %s\n", loop_dev, strerror(errno));
        return(-1);
    }

    priv_escalate();

    message(VERBOSE2, "Disassociating image from loop device\n");
    if ( ioctl(fileno(loop_fp), LOOP_CLR_FD, 0) < 0 ) {
        if ( errno != 6 ) { // Ignore loop not binded
            message(ERROR, "Could not clear loop device %s: (%d) %s\n", loop_dev, errno, strerror(errno));
            return(-1);
        }
    }

    priv_drop();

    fclose(loop_fp);

    message(DEBUG, "Returning disassociate_loop(loop_fp) = 0\n");
    return(0);
}


