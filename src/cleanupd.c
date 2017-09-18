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
    char *cleanup_dir = envar_path("SINGULARITY_CLEANUPDIR");
    char *trigger = envar_path("SINGULARITY_CLEANUPTRIGGER");
    int trigger_fd;
    int daemon_options = 0;

    singularity_message(DEBUG, "Starting cleanup process\n");

    if ( singularity_message_level() > 1 ) {
        daemon_options = 1;
    }

    if ( ( cleanup_dir == NULL ) || ( trigger == NULL ) ) {
        singularity_message(ERROR, "Environment is not properly setup\n");
        ABORT(255);
    }

    if ( is_dir(cleanup_dir) != 0 ) {
        singularity_message(ERROR, "Cleanup location is not a directory: %s\n", cleanup_dir);
        ABORT(255);
    }

    singularity_message(DEBUG, "Opening cleanup trigger file: %s\n", trigger);
    if ( ( trigger_fd = open(trigger, O_RDONLY, 00644) ) < 0 ) {
        singularity_message(ERROR, "Failed opening trigger file %s: %s\n", trigger, strerror(errno));
        ABORT(255);
    }

    singularity_message(DEBUG, "Checking to see if we need to daemonize\n");
    if ( flock(trigger_fd, LOCK_EX | LOCK_NB) != 0 ) {
        singularity_message(VERBOSE, "Daemonizing cleandir cleanup process\n");
        if ( daemon(daemon_options, daemon_options) != 0 ) {
            singularity_message(ERROR, "Failed daemonizing cleanup process: %s\n", strerror(errno));
            ABORT(255);
        }
    }

    singularity_message(DEBUG, "Waiting for exclusive flock() on trigger file descriptor: %d\n", trigger_fd);
    if ( flock(trigger_fd, LOCK_EX) == 0 ) {
        singularity_message(VERBOSE, "Cleaning directory: %s\n", cleanup_dir);
        if ( s_rmdir(cleanup_dir) < 0 ) {
            unlink(trigger);
            singularity_message(ERROR, "Could not remove directory %s: %s\n", cleanup_dir, strerror(errno));
            ABORT(255);
        }
        close(trigger_fd);
        unlink(trigger);
    }

    return(retval);
}
