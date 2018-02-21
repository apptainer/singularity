/*
 * Copyright (c) 2016, Brian Bockelman. All rights reserved.
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

#include <errno.h>
#include <fcntl.h>
#include <signal.h>
#include <string.h>
#include <unistd.h>
#include <poll.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <stdio.h>
#include <sys/file.h>
#include <sys/mount.h>

#include "util/message.h"
#include "util/util.h"
#include "util/file.h"
#include "util/registry.h"
#include "util/fork.h"
#include "util/privilege.h"

#ifndef LIBEXECDIR
#error LIBEXECDIR not defined
#endif

char *trigger = NULL;

int singularity_cleanupd(void) {
    char *cleanup_dir = singularity_registry_get("CLEANUPDIR");
    int trigger_fd = -1;

    singularity_registry_set("CLEANUPD_FD", "-1");
    
    if ( singularity_registry_get("DAEMON_JOIN") ) {
        singularity_message(ERROR, "Internal Error - This function should not be called when joining an instance\n");
    }

    if ( ( singularity_registry_get("NOSESSIONCLEANUP") != NULL ) || ( singularity_registry_get("NOCLEANUP") != NULL ) ) {
        singularity_message(DEBUG, "Not running a cleanup thread, requested not to\n");
        return(0);
    }

    if ( cleanup_dir == NULL ) {
        singularity_message(DEBUG, "Not running a cleanup thread, no 'SINGULARITY_CLEANUPDIR' defined\n");
        return(0);
    }

    if ( is_dir(cleanup_dir) != 0 ) {
        singularity_message(WARNING, "Cleanup path must be a directory: %s\n", cleanup_dir);
        return(-1);
    }

    if ( trigger == NULL ) {
        char *rand = NULL;

        if ( ( rand = random_string(8) ) == NULL ) {
            singularity_message(ERROR, "Failed obtaining a random string for temporary cleanup trigger\n");
            ABORT(255);
        }

        trigger = strjoin("/tmp/.singularity-cleanuptrigger.", rand);
        singularity_message(DEBUG, "Creating new cleanup trigger file: %s\n", trigger);

        singularity_message(DEBUG, "Opening cleanup trigger file: %s\n", trigger);
        if ( ( trigger_fd = open(trigger, O_WRONLY | O_CREAT, 00644) ) < 0 ) {
            singularity_message(ERROR, "Failed opening trigger file %s: %s\n", trigger, strerror(errno));
            ABORT(255);
        }

        singularity_message(DEBUG, "Gaining an exclusive flock on FD %d\n", trigger_fd);
        if ( flock(trigger_fd, LOCK_EX | LOCK_NB) < 0 ) {
            singularity_message(ERROR, "Could not obtain flock() on cleanup trigger file\n");
            ABORT(255);
        }

        singularity_registry_set("CLEANUPD_FD", int2str(trigger_fd));
        
    } else {
        singularity_message(DEBUG, "Using existing cleanup trigger file: %s\n", trigger);
    }

    int child = fork();
    if ( child == 0 ) {
        close(trigger_fd);

        singularity_message(VERBOSE, "Exec'ing cleanupd thread: %s\n", joinpath(LIBEXECDIR, "/singularity/bin/cleanupd"));

        envar_set("SINGULARITY_CLEANUPDIR", cleanup_dir, 1);
        envar_set("SINGULARITY_CLEANUPTRIGGER", trigger, 1);
        execl(joinpath(LIBEXECDIR, "/singularity/bin/cleanupd"), "Singularity: cleanup", NULL); // Flawfinder: ignore (on top of old smokey...)

        singularity_message(ERROR, "Exec of cleanupd process failed %s: %s\n", joinpath(LIBEXECDIR, "/singularity/bin/cleanupd"), strerror(errno));
        exit(255);

    } else if ( child > 0 ) {
        int tmpstatus;

        waitpid(child, &tmpstatus, 0);  
        if ( WEXITSTATUS(tmpstatus) != 0 ) {
            ABORT(255);
        }
    }

    return(0);
}
