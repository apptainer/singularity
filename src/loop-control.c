/* 
 * Copyright (c) 2015-2016, Gregory M. Kurtzer. All rights reserved.
 * 
 * “Singularity” Copyright (c) 2016, The Regents of the University of California,
 * through Lawrence Berkeley National Laboratory (subject to receipt of any
 * required approvals from the U.S. Dept. of Energy).  All rights reserved.
 * 
 * If you have questions about your rights to use or distribute this software,
 * please contact Berkeley Lab's Innovation & Partnerships Office at
 * IPO@lbl.gov.
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

#ifndef LO_FLAGS_AUTOCLEAR
#define LO_FLAGS_AUTOCLEAR 4
#endif

#define MAX_LOOP_DEVS 128



char * obtain_loop_dev(void) {
    char * loop_device;
    int devnum = -1;
    int i;

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
                break;
            }

        } else {
            devnum = i;
            break;
        }
    }

    if ( devnum >= 0 ) {
        loop_device = (char*) malloc(intlen(devnum) + 12);
        snprintf(loop_device, intlen(devnum) + 11, "/dev/loop%d", devnum);

        //printf("Checking for loop device: %s\n", *loop_device);
        if ( is_blk(loop_device) < 0 ) {
            //printf("Creating loop device: %s\n", *loop_device);
            if ( mknod(loop_device, S_IFBLK | 0644, makedev(7, devnum)) < 0 ) {
                fprintf(stderr, "Could not create %s: %s\n", loop_device, strerror(errno));
                return(NULL);
            }
        }
    } else {
        fprintf(stderr, "ERROR: Could not obtain a loop device number\n");
        return(NULL);
    }

    return(loop_device);
}



int associate_loop(int image_fd, int loop_fd) {
    struct loop_info64 lo64 = {0};
    struct loop_info lo = {0};
    int offset = 0;

    lo64.lo_flags = LO_FLAGS_AUTOCLEAR;
    lo64.lo_offset = offset;

    if ( ioctl(loop_fd, LOOP_SET_FD, image_fd) < 0 ) {
        fprintf(stderr, "ERROR: Failed to associate image to loop\n");
        return(-1);
    }

    if ( ioctl(loop_fd, LOOP_SET_STATUS64, &lo64) < 0 ) {
        (void)ioctl(loop_fd, LOOP_CLR_FD, 0);
        fprintf(stderr, "ERROR: Failed to set loop flags on loop device: %s\n", strerror(errno));
        return(-1);
    }

    return(0);
}


int disassociate_loop(int loop_fd) {
    if ( ioctl(loop_fd, LOOP_CLR_FD, 0) != 0 ) {
        fprintf(stderr, "ERROR: Could not clear loop device\n");
        return(-1);
    }

    return(0);
}
