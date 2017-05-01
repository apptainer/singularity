/* 
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


#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <errno.h>
#include <string.h>
#include <fcntl.h>
#include <sys/file.h>

#include "config.h"
#include "util/file.h"
#include "util/util.h"
#include "util/message.h"


int main(int argc, char **argv) {
    int retval = 0;
    int cleandir_fd;
    int cleandir_lock_fd;
    char *cleandir = envar_path("SINGULARITY_CLEANDIR");
    char *cleandir_lock = joinpath(cleandir, "/cleanup");

    singularity_message(DEBUG, "Starting cleanup process\n");

    if ( cleandir == NULL ) {
        singularity_message(ERROR, "SINGULARITY_CLEANDIR is not defined\n");
        ABORT(255);
    }

    if ( is_dir(cleandir) != 0 ) {
        singularity_message(ERROR, "SINGULARITY_CLEANDIR is not a directory: %s\n", cleandir);
        ABORT(255);
    }

    singularity_message(DEBUG, "Opening cleandir file descriptor\n");
    if ( ( cleandir_fd = open(cleandir, O_RDONLY) ) < 0 ) { // Flawfinder: ignore
        singularity_message(ERROR, "Could not obtain file descriptor on cleanup directory %s: %s\n", cleandir, strerror(errno));
        ABORT(255);
    }

    singularity_message(DEBUG, "Opening cleandir_lock file descriptor\n");
    if ( ( cleandir_lock_fd = open(cleandir_lock, O_CREAT | O_RDWR, 0755) ) < 0 ) {
        singularity_message(ERROR, "Could not obtain file descriptor for cleanup lock process %s: %s\n", cleandir_lock, strerror(errno));
        ABORT(255);
    }

    if ( flock(cleandir_lock_fd, LOCK_EX | LOCK_NB) != 0 ) {
        singularity_message(VERBOSE, "Not spawning another cleanup process,... \n");
        return(0);
    }

    singularity_message(VERBOSE, "Daemonizing cleandir cleanup process\n");
    if ( daemon(0, 0) != 0 ) {
        singularity_message(ERROR, "Failed daemonizing cleanup process: %s\n", strerror(errno));
        ABORT(255);
    }

    singularity_message(DEBUG, "Waiting for exclusive flock() on cleandir: %s\n", cleandir);

    if ( flock(cleandir_fd, LOCK_EX) == 0 ) {
        singularity_message(VERBOSE, "Cleaning directory: %s\n", cleandir);
        if ( s_rmdir(cleandir) < 0 ) {
            singularity_message(ERROR, "Could not remove directory %s: %s\n", cleandir, strerror(errno));
            ABORT(255);
        }
    }

    return(retval);
}
