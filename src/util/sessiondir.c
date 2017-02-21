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

#include "util/message.h"
#include "util/util.h"
#include "util/file.h"
#include "util/registry.h"
#include "util/config_parser.h"
#include "util/fork.h"


int singularity_sessiondir(void) {
    char *tmpdir = NULL;
    char *sessiondir = NULL;
    int sessiondir_fd;

    if ( ( sessiondir = singularity_registry_get("SESSIONDIR") ) != NULL ) {
        singularity_message(DEBUG, "Got SINGULARITY_SESSIONDIR: %s\n", tmpdir);
    } else if ( ( tmpdir = strdup(singularity_config_get_value(SESSIONDIR_PREFIX)) ) != NULL ) {
        sessiondir = strjoin(tmpdir, random_string(10));
        singularity_message(DEBUG, "Got sessiondir from configuration: '%s'\n", sessiondir);
        singularity_registry_set("SESSIONDIR", sessiondir);
    } else {
        singularity_message(ERROR, "Could not obtain session directory for process\n");
        ABORT(255);
    }

    singularity_message(VERBOSE, "Creating session directory: %s\n", sessiondir);
    if ( s_mkpath(sessiondir, 0755) < 0 ) {
        singularity_message(ERROR, "Failed creating session directory %s: %s\n", sessiondir, strerror(errno));
        ABORT(255);
    }

    singularity_message(DEBUG, "Opening sessiondir file descriptor\n");
    if ( ( sessiondir_fd = open(sessiondir, O_RDONLY) ) < 0 ) { // Flawfinder: ignore
        singularity_message(ERROR, "Could not obtain file descriptor for session directory %s: %s\n", sessiondir, strerror(errno));
        ABORT(255);
    }

    singularity_message(DEBUG, "Setting shared flock() on session directory\n");
    if ( flock(sessiondir_fd, LOCK_SH | LOCK_NB) < 0 ) {
        singularity_message(ERROR, "Could not obtain shared lock on %s: %s\n", sessiondir, strerror(errno));
        ABORT(255);
    }

    if ( ( singularity_registry_get("NOSESSIONCLEANUP") == NULL ) || ( singularity_registry_get("NOCLEANUP") == NULL ) ) {
        // singularity_fork() is currently causing problems with mvapich2, plus
        // it doesn't exec with the binaries real name properly
//        int child = singularity_fork(); 
        int child = fork();


        if ( child == 0 ) {
            singularity_message(DEBUG, "Continuing Singularity as child thread\n");
        } else if ( child > 0 ) {
            char *cleanup_proc;

            cleanup_proc = joinpath(LIBEXECDIR, "/singularity/bin/cleanupd");

            setenv("SINGULARITY_CLEANDIR", sessiondir, 1);
            close(sessiondir_fd);

            singularity_message(DEBUG, "Exec()'ing the cleanupd process\n");

            execl(joinpath(LIBEXECDIR, "/singularity/bin/cleanupd"), "singularity: cleanupd", NULL);
            //execl(cleanup_proc, cleanup_proc, NULL);
        } else {
            singularity_message(ERROR, "Could not fork cleanupd process: %s\n", strerror(errno));
            ABORT(255);
        }
    }

    return(0);
}

