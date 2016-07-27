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

#ifndef LO_FLAGS_AUTOCLEAR
#define LO_FLAGS_AUTOCLEAR 4
#endif

#define MAX_LOOP_DEVS 128


FILE *loop_attach(char *loop_dev) {
    FILE *loop_fp;
    if ( ( loop_fp = fopen(loop_dev, "r+") ) == NULL ) { // Flawfinder: ignore (not user modifyable)
        message(VERBOSE, "Could not open loop device %s: %s\n", loop_dev, strerror(errno));
        ABORT(255);
        return(NULL);
    }

    return(loop_fp);
}


FILE *loop_bind(FILE *image_fp, char **loop_dev, int autoclear) {
    struct loop_info64 lo64 = {0};
    FILE *loop_fp;
    int i;

    message(DEBUG, "Called loop_bind(image_fp, **{loop_dev)\n");

    if ( autoclear > 0 ) {
        lo64.lo_flags = LO_FLAGS_AUTOCLEAR;
    }
    lo64.lo_offset = image_offset(image_fp);

    for( i=0; i < MAX_LOOP_DEVS; i++ ) {
        char *test_loopdev = strjoin("/dev/loop", int2str(i));

        if ( is_blk(test_loopdev) < 0 ) {
            message(VERBOSE, "Creating loop device: %s\n", test_loopdev);
            if ( mknod(test_loopdev, S_IFBLK | 0644, makedev(7, i)) < 0 ) {
                message(ERROR, "Could not create %s: %s\n", test_loopdev, strerror(errno));
                ABORT(255);
            }
        }

        if ( ( loop_fp = fopen(test_loopdev, "r+") ) == NULL ) { // Flawfinder: ignore (not user modifyable)
            message(VERBOSE, "Could not open loop device %s: %s\n", test_loopdev, strerror(errno));
            continue;
        }

        message(VERBOSE2, "Attempting to associate image pointer to loop device\n");
        if ( ioctl(fileno(loop_fp), LOOP_SET_FD, fileno(image_fp)) < 0 ) {
            if ( errno == 16 ) {
                message(VERBOSE3, "Loop device is in use: %s\n", test_loopdev);
                fclose(loop_fp);
                continue;
            } else {
                message(WARNING, "Could not associate image to loop %s: %s\n", test_loopdev, strerror(errno));
                fclose(loop_fp);
                continue;
            }
        }

        message(VERBOSE, "Found valid loop device: %s\n", test_loopdev);

        message(VERBOSE2, "Setting loop device flags\n");
        if ( ioctl(fileno(loop_fp), LOOP_SET_STATUS64, &lo64) < 0 ) {
            fprintf(stderr, "ERROR: Failed to set loop flags on loop device: %s\n", strerror(errno));
            (void)ioctl(fileno(loop_fp), LOOP_CLR_FD, 0);
            (void)loop_free(*loop_dev);
            ABORT(255);
        }
        *loop_dev = strdup(test_loopdev);

        message(VERBOSE, "Using loop device: %s\n", *loop_dev);

        message(DEBUG, "Returning loop_bind(image_fp) = loop_fp\n");

        return(loop_fp);
    }

    message(ERROR, "No valid loop devices available\n");
    ABORT(255);

    return(NULL);
}


int loop_free(char *loop_dev) {
    FILE *loop_fp;

    message(DEBUG, "Called loop_free(%s)\n", loop_dev);

    if ( is_blk(loop_dev) < 0 ) {
        message(ERROR, "Loop device is not a valid block device: %s\n", loop_dev);
        ABORT(255);
    }

    if ( ( loop_fp = fopen(loop_dev, "r") ) == NULL ) { // Flawfinder: ignore (only opening read only, and must be a block device)
        message(VERBOSE, "Could not open loop device %s: %s\n", loop_dev, strerror(errno));
        return(-1);
    }

    message(DEBUG, "Called disassociate_loop(loop_fp)\n");

    message(VERBOSE2, "Disassociating image from loop device\n");
    if ( ioctl(fileno(loop_fp), LOOP_CLR_FD, 0) < 0 ) {
        if ( errno != 6 ) { // Ignore loop not binded
            message(ERROR, "Could not clear loop device %s: (%d) %s\n", loop_dev, errno, strerror(errno));
            return(-1);
        }
    }

    message(DEBUG, "Returning disassociate_loop(loop_fp) = 0\n");
    return(0);
}


// Leaving the below code intact for comparasion and reference
/*
char * obtain_loop_dev(void) {
    char * loop_device;
    int devnum = -1;
    int i;

    message(DEBUG, "Called obtain_loop_dev(void)\n");

    // We brute force this to be compatible with older loop implementations
    // that don't provide /dev/loop-control
    for( i=0; i < MAX_LOOP_DEVS; i++ ) {
        char *test_loopdev = strjoin("/dev/loop", int2str(i));
        struct loop_info loop_status = {0};
        int loop_fd;

        if ( ( loop_fd = open(test_loopdev, O_RDONLY) ) >= 0 ) {
            int ret = ioctl(loop_fd, LOOP_GET_STATUS, &loop_status);
            close(loop_fd);
            if ( ret != 0 ) {
                devnum = i;
                message(DEBUG, "Found available existing loop device number: %d\n", devnum);
                break;
            }

        } else {
            devnum = i;
            message(DEBUG, "Found new loop device number: %d\n", devnum);
            break;
        }
    }

    if ( devnum >= 0 ) {
        loop_device = (char*) malloc(intlen(devnum) + 12);
        snprintf(loop_device, intlen(devnum) + 11, "/dev/loop%d", devnum);

        message(VERBOSE, "Using loop device: %s\n", loop_device);

        if ( is_blk(loop_device) < 0 ) {
            message(VERBOSE, "Creating loop device: %s\n", loop_device);
            if ( mknod(loop_device, S_IFBLK | 0644, makedev(7, devnum)) < 0 ) {
                message(ERROR, "Could not create %s: %s\n", loop_device, strerror(errno));
                ABORT(255);
            }
        }
    } else {
        message(ERROR, "Could not obtain a loop device number\n");
        ABORT(255);
    }

    message(DEBUG, "Returning obtain_loop_dev(void) = %s\n", loop_device);
    return(loop_device);
}



int associate_loop(FILE *image_fp, FILE *loop_fp, int autoclear) {
    struct loop_info64 lo64 = {0};
    int image_fd = fileno(image_fp);
    int loop_fd = fileno(loop_fp);

    message(DEBUG, "Called associate_loop(image_fp, loop_fp, %d)\n", autoclear);

    if ( autoclear > 0 ) {
        message(DEBUG, "Setting loop flags to LO_FLAGS_AUTOCLEAR\n");
        lo64.lo_flags = LO_FLAGS_AUTOCLEAR;
    }
    lo64.lo_offset = image_offset(image_fp);

    message(DEBUG, "Setting image offset to: %d\n", lo64.lo_offset);

    message(VERBOSE2, "Associating image to loop device\n");
    if ( ioctl(loop_fd, LOOP_SET_FD, image_fd) < 0 ) {
        fprintf(stderr, "ERROR: Failed to associate image to loop (%d): %s\n", errno, strerror(errno));
        ABORT(255);
    }

    message(VERBOSE2, "Setting loop device flags\n");
    if ( ioctl(loop_fd, LOOP_SET_STATUS64, &lo64) < 0 ) {
        (void)ioctl(loop_fd, LOOP_CLR_FD, 0);
        fprintf(stderr, "ERROR: Failed to set loop flags on loop device: %s\n", strerror(errno));
        (void)disassociate_loop(loop_fp);
        ABORT(255);
    }

    message(DEBUG, "Returning associate_loop(image_fp, loop_fp, %d) = 0\n", autoclear);
    return(0);
}


int disassociate_loop(FILE *loop_fp) {
    int loop_fd = fileno(loop_fp);

    message(DEBUG, "Called disassociate_loop(loop_fp)\n");

    message(VERBOSE2, "Disassociating image from loop device\n");
    if ( ioctl(loop_fd, LOOP_CLR_FD, 0) != 0 ) {
        message(ERROR, "Could not clear loop device: %s\n", strerror(errno));
        ABORT(255);
    }

    message(DEBUG, "Returning disassociate_loop(loop_fp) = 0\n");
    return(0);
}
*/
