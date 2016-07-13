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

#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/file.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <sys/param.h>
#include <errno.h> 
#include <string.h>
#include <fcntl.h>  

#include "config.h"
#include "container_actions.h"
#include "util.h"
#include "file.h"
#include "message.h"


int container_run(int argc, char **argv) {
    message(DEBUG, "Called container_run(%d, **argv)\n", argc);
    if ( is_exec("/singularity") == 0 ) {
        argv[0] = strdup("/singularity");
        message(VERBOSE, "Found /singularity inside container, exec()'ing...\n");
        if ( execv("/singularity", argv) != 0 ) { // Flawfinder: ignore (exec* is necessary)
            message(ERROR, "Exec of /bin/sh failed: %s\n", strerror(errno));
            ABORT(255);
        }
    } else {
        message(WARNING, "No Singularity runscript found, launching 'shell'\n");
        container_shell(argc, argv);
    }

    message(ERROR, "We should not have reached the end of container_run()\n");
    return(-1);
}

int container_exec(int argc, char **argv) {
    message(DEBUG, "Called container_exec(%d, **argv)\n", argc);
    if ( argc <= 1 ) {
        message(ERROR, "Exec requires a command to run\n");
        ABORT(255);
    }

    message(VERBOSE, "Exec'ing program: %s\n", argv[1]);
    if ( execvp(argv[1], &argv[1]) != 0 ) { // Flawfinder: ignore (exec* is necessary)
        message(ERROR, "execvp of '%s' failed: %s\n", argv[1], strerror(errno));
        ABORT(255);
    }

    message(ERROR, "We should not have reached the end of container_exec\n");
    return(-1);
}

int container_shell(int argc, char **argv) {
    message(DEBUG, "Called container_shell(%d, **argv)\n", argc);

    if ( is_exec("/bin/bash") == 0 ) {
        char *args[argc+2]; // Flawfinder: ignore
        int i;

        message(VERBOSE, "Found /bin/bash, setting arguments --norc and --noprofile\n");

        args[0] = strdup("/bin/bash");
        args[1] = strdup("--norc");
        args[2] = strdup("--noprofile");
        for(i=1; i<=argc; i++) {
            args[i+2] = argv[i];
        }

        message(VERBOSE, "Exec()'ing /bin/bash...\n");
        if ( execv("/bin/bash", args) != 0 ) { // Flawfinder: ignore (exec* is necessary)
            message(ERROR, "Exec of /bin/bash failed: %s\n", strerror(errno));
        }
    } else {
        argv[0] = strdup("/bin/sh");
        message(VERBOSE, "Exec()'ing /bin/sh...\n");
        if ( execv("/bin/sh", argv) != 0 ) { // Flawfinder: ignore (exec* is necessary)
            message(ERROR, "Exec of /bin/sh failed: %s\n", strerror(errno));
        }
    }

    message(ERROR, "We should not have reached the end of container_shell()\n");
    return(-1);
}


int container_daemon_start(char *sessiondir) {
    FILE *comm;
    char line[256]; // Flawfinder: ignore (this is hard limit in fgets() below)

    message(DEBUG, "Called container_daemon_start(%s)\n", sessiondir);

// TODO: Create a daemon_start_init function
    message(DEBUG, "Opening daemon.comm for writing\n");
    if ( ( comm = fopen(joinpath(sessiondir, "daemon.comm"), "r") ) == NULL ) { // Flawfinder: ignore
        message(ERROR, "Could not open communication fifo %s: %s\n", joinpath(sessiondir, "daemon.comm"), strerror(errno));
        ABORT(255);
    }

    message(DEBUG, "Waiting for read on daemon.comm\n");
    while ( fgets(line, 256, comm) ) {
        if ( strcmp(line, "stop") == 0 ) {
            message(INFO, "Stopping daemon\n");
            break;
        } else {
            message(WARNING, "Got unsupported daemon.comm command: '%s'\n", line);
        }
    }
    fclose(comm);

    message(DEBUG, "Return container_daemon_start(%s) = 0\n", sessiondir);
    return(0);
}


int container_daemon_stop(char *sessiondir) {
    FILE *comm;
    FILE *test_daemon_fp;
    int daemon_fd;

    message(DEBUG, "Called container_daemon_stop(%s)\n", sessiondir);

    message(VERBOSE, "Checking if daemon is currently running for this container\n");
    if ( is_file(joinpath(sessiondir, "daemon.pid")) < 0 ) {
        message(ERROR, "Daemon process is not running\n");
        return(0);
    }

    message(DEBUG, "Opening daemon.pid for reading\n");
    if ( ( test_daemon_fp = fopen(joinpath(sessiondir, "daemon.pid"), "r") ) == NULL ) { // Flawfinder: ignore
        message(ERROR, "Could not open daemon pid file %s: %s\n", joinpath(sessiondir, "daemon.pid"), strerror(errno));
        ABORT(255);
    }

    message(DEBUG, "Testing to see if daemon process is still active\n");
    daemon_fd = fileno(test_daemon_fp);
    if ( flock(daemon_fd, LOCK_SH | LOCK_NB) == 0 ) {
        message(INFO, "No active container daemon active\n");
        return(0);
    }

    message(DEBUG, "Connecting to daemon.comm FIFO\n");
    if ( is_fifo(joinpath(sessiondir, "daemon.comm")) < 0 ) {
        message(ERROR, "Container daemon COMM not available\n");
        ABORT(255);
    }

    message(VERBOSE, "Opening daemon.comm for writing\n");
    if ( ( comm = fopen(joinpath(sessiondir, "daemon.comm"), "w") ) == NULL ) { //Flawfinder: ignore
        message(ERROR, "Could not open fifo for writing %s: %s\n", joinpath(sessiondir, "daemon.comm"), strerror(errno));
        ABORT(255);
    }

    message(VERBOSE, "Sending stop command to daemon process\n");
    fputs("stop", comm);

    fclose(comm);

    message(DEBUG, "Return container_daemon_stop(%s) = 0\n", sessiondir);
    return(0);
}



