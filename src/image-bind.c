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

#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/mount.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <sys/param.h>
#include <errno.h> 
#include <signal.h>
#include <sched.h>
#include <string.h>
#include <fcntl.h>  
#include <grp.h>
#include <libgen.h>

#include "config.h"
#include "mounts.h"
#include "file.h"
#include "util.h"
#include "loop-control.h"
#include "message.h"


int main(int argc, char ** argv) {
    uid_t uid = geteuid();

    if ( argv[1] == NULL || argv[2] == NULL ) {
        fprintf(stderr, "USAGE: %s [attach/detach] [image/loop]\n", argv[0]);
        return(1);
    }

    message(VERBOSE, "Checking calling user\n");
    if ( uid != 0 ) {
        message(ERROR, "Calling user must be root\n");
        ABORT(1);
    }

    message(VERBOSE, "Checking command: %s\n", argv[1]);
    if ( strcmp(argv[1], "attach") == 0 ) {
        FILE *loop_fp;
        FILE *containerimage_fp;
        char *containerimage;
        char *loop_dev;

        message(VERBOSE, "Preparing to attach container to loop\n");

        containerimage = strdup(argv[2]);

        message(VERBOSE, "Evaluating image: %s\n", containerimage);
    
        message(VERBOSE, "Checking if container image exists\n");
        if ( is_file(containerimage) < 0 ) {
            message(ERROR, "Container image not found: %s\n", containerimage);
            ABORT(1);
        }

        message(VERBOSE, "Checking if container can be opened read/write\n");
        if ( ( containerimage_fp = fopen(containerimage, "r+") ) < 0 ) { // Flawfinder: ignore
            message(ERROR, "Could not open image %s: %s\n", containerimage, strerror(errno));
            ABORT(255);
        }

        message(DEBUG, "Binding container to loop interface\n");
        if ( ( loop_fp = loop_bind(containerimage_fp, &loop_dev, 0)) == NULL ) {
            message(ERROR, "Could not bind image to loop!\n");
            ABORT(255);
        }

        printf("%s\n", loop_dev);
    } else if (strcmp(argv[1], "detach") == 0 ) {
        char *loop_dev;

        loop_dev = strdup(argv[2]);

        message(VERBOSE, "Preparing to detach loop: %s\n", loop_dev);

        message(VERBOSE, "Checking loop device\n");
        if ( is_blk(loop_dev) < 0 ) {
            message(ERROR, "Block device not found: %s\n", loop_dev);
            ABORT(255);
        }

        message(VERBOSE, "Unbinding container image from loop\n");
        if ( loop_free(loop_dev) < 0 ) {
            message(ERROR, "Failed to detach loop device: %s\n", loop_dev);
            ABORT(255);
        }

    }

    return(0);
}
