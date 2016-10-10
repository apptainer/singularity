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

#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <string.h>
#include <sys/file.h>
#include <sys/stat.h>
#include <unistd.h>
#include <stdlib.h>

#include "util/file.h"
#include "util/util.h"
#include "lib/message.h"
#include "lib/privilege.h"
#include "lib/sessiondir.h"

int daemon_fd; // Global to stay defined through the life of the process
pid_t parent_daemon;

void action_start_init(void) {
#ifdef NO_SETNS
    singularity_message(ERROR, "This host does not support joining existing name spaces\n");
    ABORT(1);
#endif
    singularity_message(VERBOSE, "Namespace daemon function requested\n");

    singularity_message(DEBUG, "Forking daemon process into the background\n");
    if ( daemon(0, 0) < 0 ) {
        singularity_message(ERROR, "Could not daemonize: %s\n", strerror(errno));
        ABORT(255);
    }

    parent_daemon = getpid();

}


void action_start_do(int argc, char **argv) {
    FILE *comm;
    char *line = (char *) malloc(256);
    char *sessiondir = singularity_sessiondir_get();

    if ( ( daemon_fd = open(joinpath(sessiondir, "daemon.pid"), O_CREAT | O_RDWR, 0755) ) < 0 ) { // Flawfinder: ignore
        singularity_message(ERROR, "Could not open daemon pid file for writing %s: %s\n", joinpath(sessiondir, "daemon.pid"), strerror(errno));
        ABORT(255);
    }

    if ( flock(daemon_fd, LOCK_EX | LOCK_NB) != 0 ) {
        singularity_message(ERROR, "Could not obtain lock, another daemon process running?\n");
        ABORT(255);
    }

    if ( write(daemon_fd, int2str(parent_daemon), intlen(parent_daemon)) <= 0 ) {
        singularity_message(ERROR, "Could not write PID to pidfile: %s\n", strerror(errno));
        ABORT(255);
    }
    if ( fsync(daemon_fd) < 0 ) {
        singularity_message(ERROR, "Could not flush PID to pidfile: %s\n", strerror(errno));
        ABORT(255);
    }

    singularity_message(VERBOSE, "Creating daemon.comm fifo\n");
    if ( is_fifo(joinpath(sessiondir, "daemon.comm")) < 0 ) {
        if ( mkfifo(joinpath(sessiondir, "daemon.comm"), 0664) < 0 ) {
            singularity_message(ERROR, "Could not create communication fifo: %s\n", strerror(errno));
            ABORT(255);
        }
    }

    singularity_message(DEBUG, "Opening daemon.comm for reading\n");
    if ( ( comm = fopen(joinpath(sessiondir, "daemon.comm"), "r") ) == NULL ) { // Flawfinder: ignore
        singularity_message(ERROR, "Could not open communication fifo %s: %s\n", joinpath(sessiondir, "daemon.comm"), strerror(errno));
        ABORT(255);
    }

    singularity_message(DEBUG, "Waiting for read on daemon.comm\n");
    while ( fgets(line, 255, comm) ) {
        if ( strcmp(line, "stop") == 0 ) {
            singularity_message(INFO, "Stopping daemon\n");
            break;
        } else {
            singularity_message(WARNING, "Got unsupported daemon.comm command: '%s'\n", line);
        }
    }
    fclose(comm);

    singularity_message(VERBOSE, "Namespace process exiting...\n");
    exit(0);
}
