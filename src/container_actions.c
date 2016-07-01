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
        if ( execv("/singularity", argv) != 0 ) {
            message(ERROR, "Exec of /bin/sh failed: %s\n", strerror(errno));
            ABORT(255);
        }
    } else {
        message(WARNING, "No Singularity runscript found, launching 'shell'\n");
        container_shell(argc, argv);
    }

    message(ERROR, "We should not have reached here...\n");
    return(-1);
}

int container_exec(int argc, char **argv) {
    message(DEBUG, "Called container_exec(%d, **argv)\n", argc);
    if ( argc <= 1 ) {
        message(ERROR, "Exec requires a command to run\n");
        ABORT(255);
    }

    message(VERBOSE, "Exec'ing program: %s\n", argv[1]);
    if ( execvp(argv[1], &argv[1]) != 0 ) {
        message(ERROR, "execvp of '%s' failed: %s\n", argv[1], strerror(errno));
        ABORT(255);
    }

    message(ERROR, "We should not have reached here...\n");
    return(-1);
}

int container_shell(int argc, char **argv) {
    message(DEBUG, "Called container_shell(%d, **argv)\n", argc);

    if ( is_exec("/bin/bash") == 0 ) {
        char *args[argc+2];
        int i;

        message(VERBOSE, "Found /bin/bash, setting arguments --norc and --noprofile\n");

        args[0] = strdup("/bin/bash");
        args[1] = strdup("--norc");
        args[2] = strdup("--noprofile");
        for(i=1; i<=argc; i++) {
            args[i+2] = argv[i];
        }

        message(VERBOSE, "Exec()'ing /bin/bash...\n");
        if ( execv("/bin/bash", args) != 0 ) {
            message(ERROR, "Exec of /bin/bash failed: %s\n", strerror(errno));
        }
    } else {
        argv[0] = strdup("/bin/sh");
        message(VERBOSE, "Exec()'ing /bin/sh...\n");
        if ( execv("/bin/sh", argv) != 0 ) {
            message(ERROR, "Exec of /bin/sh failed: %s\n", strerror(errno));
        }
    }

    message(ERROR, "We should not have reached here...\n");
    return(-1);
}


int container_daemon_start(char *tmpdir) {
    FILE *comm;
    char line[256];

    if ( ( comm = fopen(joinpath(tmpdir, "daemon.comm"), "r") ) == NULL ) {
        message(ERROR, "Could not open communication fifo %s: %s\n", joinpath(tmpdir, "daemon.comm"), strerror(errno));
        ABORT(255);
    }

    while ( fgets(line, 256, comm) ) {
        if ( strcmp(line, "stop") == 0 ) {
            message(INFO, "Stopping daemon\n");
            break;
        }
    }
    fclose(comm);

    return(0);
}


int container_daemon_stop(char *tmpdir) {
    FILE *comm;
    FILE *test_daemon_fp;
    int daemon_fd;

    if ( is_file(joinpath(tmpdir, "daemon.pid")) < 0 ) {
        message(ERROR, "Daemon process is not running\n");
        return(0);
    }

    if ( ( test_daemon_fp = fopen(joinpath(tmpdir, "daemon.pid"), "r") ) == NULL ) {
        message(ERROR, "Could not open daemon pid file %s: %s\n", joinpath(tmpdir, "daemon.pid"), strerror(errno));
        ABORT(255);
    }

    daemon_fd = fileno(test_daemon_fp);
    if ( flock(daemon_fd, LOCK_SH | LOCK_NB) == 0 ) {
        message(INFO, "No active container daemon active\n");
        return(0);
    }

    if ( is_fifo(joinpath(tmpdir, "daemon.comm")) < 0 ) {
        message(ERROR, "Container daemon COMM not available\n");
        ABORT(255);
    }

    if ( ( comm = fopen(joinpath(tmpdir, "daemon.comm"), "w") ) == NULL ) {
        message(ERROR, "Could not open fifo for writing %s: %s\n", joinpath(tmpdir, "daemon.comm"), strerror(errno));
        ABORT(255);
    }

    fputs("stop", comm);

    fclose(comm);

    return(0);
}



