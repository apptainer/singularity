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
#include "loop-control.h"


int main(int argc, char ** argv) {
    FILE *loop_fp;
    FILE *containerimage_fp;
    char *containerimage;
    char *loop_dev;
    uid_t uid = geteuid();

    if ( uid != 0 ) {
        fprintf(stderr, "ABORT: Calling user must be root\n");
        return(1);
    }

    if ( argv[1] == NULL ) {
        fprintf(stderr, "USAGE: %s [singularity container image] [mount point]\n", argv[0]);
        return(1);
    }

    containerimage = strdup(argv[1]);

    if ( is_file(containerimage) < 0 ) {
        fprintf(stderr, "ABORT: Container image not found: %s\n", containerimage);
        return(1);
    }

    if ( ( containerimage_fp = fopen(containerimage, "r+") ) < 0 ) {
        fprintf(stderr, "ERROR: Could not open image %s: %s\n", containerimage, strerror(errno));
        return(255);
    }

    loop_dev = obtain_loop_dev();

    if ( ( loop_fp = fopen(loop_dev, "r+") ) < 0 ) {
        fprintf(stderr, "ERROR: Failed to open loop device %s: %s\n", loop_dev, strerror(errno));
        return(-1);
    }

    if ( associate_loop(containerimage_fp, loop_fp, 0) < 0 ) {
        fprintf(stderr, "ERROR: Could not associate %s to loop device %s\n", containerimage, loop_dev);
        return(255);
    }

    printf("%s\n", loop_dev);

    return(0);
}
