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
#include <linux/limits.h>

#include "config.h"
#include "lib/singularity.h"
#include "util/file.h"
#include "util/util.h"


int main(int argc, char ** argv) {
    uid_t uid = geteuid();
    FILE *containerimage_fp;
    char *containerimage;
    char *loop_dev;

    if ( argv[1] == NULL ) {
        fprintf(stderr, "USAGE: SINGULARITY_IMAGE=[image] %s [command...]\n", argv[0]);
        return(1);
    }

    singularity_message(VERBOSE, "Checking calling user\n");
    if ( uid != 0 ) {
        singularity_message(ERROR, "Calling user must be root\n");
        ABORT(1);
    }

    if ( ( containerimage = envar_path("SINGULARITY_IMAGE") ) == NULL ) {
        singularity_message(ERROR, "SINGULARITY_IMAGE not defined!\n");
        ABORT(255);
    }

    singularity_message(VERBOSE, "Evaluating image: %s\n", containerimage);

    singularity_message(VERBOSE, "Checking if container image exists\n");
    if ( is_file(containerimage) < 0 ) {
        singularity_message(ERROR, "Container image not found: %s\n", containerimage);
        ABORT(1);
    }

    singularity_message(VERBOSE, "Checking if container can be opened read/write\n");
    if ( ( containerimage_fp = fopen(containerimage, "r+") ) == NULL ) { // Flawfinder: ignore
        singularity_message(ERROR, "Could not open image %s: %s\n", containerimage, strerror(errno));
        ABORT(255);
    }

    singularity_priv_init();
    singularity_config_open(joinpath(SYSCONFDIR, "/singularity/singularity.conf"));
    singularity_sessiondir_init(containerimage);
    singularity_ns_mnt_unshare();

    free(containerimage);

    singularity_message(DEBUG, "Binding container to loop interface\n");
    if ( ( loop_dev = singularity_loop_bind(containerimage_fp)) == NULL ) {
        singularity_message(ERROR, "Could not bind image to loop!\n");
        ABORT(255);
    }

    singularity_message(VERBOSE, "Setting SINGULARITY_LOOPDEV to '%s'\n", loop_dev);
    setenv("SINGULARITY_LOOPDEV", loop_dev, 1);

    return(singularity_fork_exec(&argv[1]));
}
