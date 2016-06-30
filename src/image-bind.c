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

    if ( uid != 0 ) {
        message(ERROR, "Calling user must be root\n");
        ABORT(1);
    }

    if ( argv[1] == NULL || argv[2] == NULL ) {
        fprintf(stderr, "USAGE: %s [attach/detach] [image/loop]\n", argv[0]);
        return(1);
    }

    if ( strcmp(argv[1], "attach") == 0 ) {
        FILE *loop_fp;
        FILE *containerimage_fp;
        char *containerimage;
        char *loop_dev;
    
        containerimage = strdup(argv[2]);

        if ( is_file(containerimage) < 0 ) {
            message(ERROR, "Container image not found: %s\n", containerimage);
            ABORT(1);
        }

        if ( ( containerimage_fp = fopen(containerimage, "r+") ) < 0 ) {
            message(ERROR, "Could not open image %s: %s\n", containerimage, strerror(errno));
            ABORT(255);
        }

        loop_dev = obtain_loop_dev();

        if ( ( loop_fp = fopen(loop_dev, "r+") ) < 0 ) {
            message(ERROR, "Failed to open loop device %s: %s\n", loop_dev, strerror(errno));
            ABORT(255);
        }

        if ( associate_loop(containerimage_fp, loop_fp, 0) < 0 ) {
            message(ERROR, "Could not associate %s to loop device %s\n", containerimage, loop_dev);
            ABORT(255);
        }

        printf("%s\n", loop_dev);
    } else if (strcmp(argv[1], "detach") == 0 ) {
        FILE *loop_fp;
        char *loop_dev;
        
        loop_dev = strdup(argv[2]);

        if ( is_blk(loop_dev) < 0 ) {
            message(ERROR, "Block device not found: %s\n", loop_dev);
            ABORT(255);
        }

        if ( ( loop_fp = fopen(loop_dev, "r+") ) < 0 ) {
            message(ERROR, "Failed to open loop device %s: %s\n", loop_dev, strerror(errno));
            ABORT(255);
        }

        if ( disassociate_loop(loop_fp) < 0 ) {
            message(ERROR, "Failed to detach loop device: %s\n", loop_dev);
            ABORT(255);
        }

    }

    return(0);
}
