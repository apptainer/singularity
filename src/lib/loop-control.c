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
#include <sys/file.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <errno.h> 
#include <string.h>
#include <fcntl.h>
#include <sys/ioctl.h>

#include "config.h"
#include "lib/loop-control.h"
#include "lib/image-util.h"
#include "util/util.h"
#include "util/file.h"
//#include "image.h"
#include "lib/message.h"
#include "lib/privilege.h"
#include "sessiondir.h"

#ifndef LO_FLAGS_AUTOCLEAR
#define LO_FLAGS_AUTOCLEAR 4
#endif

#define MAX_LOOP_DEVS 128

char *loop_dev;
FILE *loop_fp;
int image_loop_file_fd; // This has to be global for the flock to be held

char *singularity_loop_bind(FILE *image_fp) {
    char *sessiondir = singularity_sessiondir_get();
    char *image_loop_file = joinpath(sessiondir, "image_loop_dev");
    struct loop_info64 lo64 = {0};
    int i;

    if ( image_fp == NULL ) {
        singularity_message(ERROR, "Called singularity_loop_bind() with NULL image pointer\n");
        ABORT(255);
    }

    singularity_message(DEBUG, "Opening image loop device file: %s\n", image_loop_file);
    if ( ( image_loop_file_fd = open(image_loop_file, O_CREAT | O_RDWR, 0644) ) < 0 ) { // Flawfinder: ignore
        singularity_message(ERROR, "Could not open image loop device cache file %s: %s\n", image_loop_file, strerror(errno));
        ABORT(255);
    }

    singularity_message(DEBUG, "Requesting exclusive flock() on loop_dev lockfile\n");
    if ( flock(image_loop_file_fd, LOCK_EX | LOCK_NB) < 0 ) {
        char *active_loop_dev;
        singularity_message(VERBOSE2, "Did not get exclusive lock on image loop device cache, assuming it is active\n");

        singularity_message(DEBUG, "Waiting to obtain shared lock on loop_dev lockfile\n");
        flock(image_loop_file_fd, LOCK_SH);

        singularity_message(DEBUG, "Obtaining cached loop device name\n");
        if ( ( active_loop_dev = filecat(image_loop_file) ) == NULL ) {
            singularity_message(ERROR, "Could not retrieve active loop device from %s\n", image_loop_file);
            ABORT(255);
        }

        singularity_message(DEBUG, "Returning with active loop device name: %s\n", active_loop_dev);
        return(active_loop_dev);
    }


#ifdef LO_FLAGS_AUTOCLEAR
    lo64.lo_flags = LO_FLAGS_AUTOCLEAR;
#endif

    singularity_message(DEBUG, "Calculating image offset\n");
    if ( ( lo64.lo_offset = singularity_image_offset(image_fp) ) < 0 ) {
        singularity_message(ERROR, "Could not obtain message offset of image\n");
        ABORT(255);
    }

    singularity_priv_escalate();
    singularity_message(DEBUG, "Finding next available loop device...\n");
    for( i=0; i < MAX_LOOP_DEVS; i++ ) {
        char *test_loopdev = strjoin("/dev/loop", int2str(i));

        if ( is_blk(test_loopdev) < 0 ) {
            if ( mknod(test_loopdev, S_IFBLK | 0644, makedev(7, i)) < 0 ) {
                singularity_message(ERROR, "Could not create %s: %s\n", test_loopdev, strerror(errno));
                ABORT(255);
            }
        }

        if ( ( loop_fp = fopen(test_loopdev, "r+") ) == NULL ) { // Flawfinder: ignore
            singularity_message(VERBOSE, "Could not open loop device %s: %s\n", test_loopdev, strerror(errno));
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
                singularity_message(WARNING, "Could not associate image to loop %s: %s\n", test_loopdev, strerror(errno));
                fclose(loop_fp);
                continue;
            }
        }

    }

    singularity_message(VERBOSE, "Found available loop device: %s\n", loop_dev);

    singularity_message(DEBUG, "Setting loop device flags\n");
    if ( ioctl(fileno(loop_fp), LOOP_SET_STATUS64, &lo64) < 0 ) {
        singularity_message(ERROR, "Failed to set loop flags on loop device: %s\n", strerror(errno));
        (void)ioctl(fileno(loop_fp), LOOP_CLR_FD, 0);
        (void)singularity_loop_free(loop_dev);
        ABORT(255);
    }

    singularity_priv_drop();

    singularity_message(VERBOSE, "Using loop device: %s\n", loop_dev);

    singularity_message(DEBUG, "Writing active loop device name (%s) to loop file cache: %s\n", loop_dev, image_loop_file);
    if ( fileput(image_loop_file, loop_dev) < 0 ) {
        singularity_message(ERROR, "Could not write to image_loop_file %s: %s\n", image_loop_file, strerror(errno));
        ABORT(255);
    }

    singularity_message(DEBUG, "Resetting exclusive flock() to shared on image_loop_file\n");
    flock(image_loop_file_fd, LOCK_SH | LOCK_NB);

    singularity_message(DEBUG, "Returning singularity_loop_bind(image_fp) = loop_fp\n");

    return(loop_dev);
}


int singularity_loop_free(char *loop_name) {

    singularity_message(DEBUG, "Called singularity_loop_free(%s)\n", loop_name);

    if ( is_blk(loop_name) < 0 ) {
        singularity_message(ERROR, "Loop device is not a valid block device: %s\n", loop_name);
        ABORT(255);
    }

    if ( ( loop_fp = fopen(loop_name, "r") ) == NULL ) { // Flawfinder: ignore
        singularity_message(VERBOSE, "Could not open loop device %s: %s\n", loop_name, strerror(errno));
        return(-1);
    }

    singularity_priv_escalate();

    singularity_message(VERBOSE2, "Disassociating image from loop device\n");
    if ( ioctl(fileno(loop_fp), LOOP_CLR_FD, 0) < 0 ) {
        if ( errno != 6 ) { // Ignore loop not binded
            singularity_message(ERROR, "Could not clear loop device %s: (%d) %s\n", loop_name, errno, strerror(errno));
            return(-1);
        }
    }

    singularity_priv_drop();

    fclose(loop_fp);

    singularity_message(DEBUG, "Returning disassociate_loop(loop_fp) = 0\n");
    return(0);
}


