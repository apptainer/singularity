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
#include "util.h"
#include "file.h"


int container_daemon_start(char *tmpdir);
int container_daemon_stop(char *tmpdir);
int container_shell(int argc, char **argv);
int container_exec(int argc, char **argv);
int container_run(int argc, char **argv);


int container_run(int argc, char **argv) {
    if ( is_exec("/singularity") == 0 ) {
        argv[0] = strdup("/singularity");
        if ( execv("/singularity", argv) != 0 ) {
            fprintf(stderr, "ABORT: exec of /bin/sh failed: %s\n", strerror(errno));
        }
    } else {
        fprintf(stderr, "No Singularity runscript found, launching 'shell'\n");
        container_shell(argc, argv);
    }

    return(-1);
}

int container_exec(int argc, char **argv) {
    if ( argc <= 1 ) {
        fprintf(stderr, "ABORT: Exec requires a command to run\n");
        return(-1);
    }
    if ( execvp(argv[1], &argv[1]) != 0 ) {
        fprintf(stderr, "ABORT: execvp of '%s' failed: %s\n", argv[1], strerror(errno));
        return(-1);
    }

    return(-1);
}

int container_shell(int argc, char **argv) {

    if ( is_exec("/bin/bash") == 0 ) {
        char *args[argc+2];
        int i;

        args[0] = strdup("/bin/bash");
        args[1] = strdup("--norc");
        args[2] = strdup("--noprofile");
        for(i=1; i<=argc; i++) {
            args[i+2] = argv[i];
        }

        if ( execv("/bin/bash", args) != 0 ) {
            fprintf(stderr, "ABORT: exec of /bin/bash failed: %s\n", strerror(errno));
        }
    } else {
        argv[0] = strdup("/bin/sh");
        if ( execv("/bin/sh", argv) != 0 ) {
            fprintf(stderr, "ABORT: exec of /bin/sh failed: %s\n", strerror(errno));
        }
    }

    return(-1);
}


int container_daemon_start(char *tmpdir) {
    FILE *comm;
    char line[256];

    if ( ( comm = fopen(joinpath(tmpdir, "daemon.comm"), "r") ) == NULL ) {
        fprintf(stderr, "Could not open fifo %s: %s\n", joinpath(tmpdir, "daemon.comm"), strerror(errno));
        return(-1);
    }

    if ( chdir("/") < 0 ) {
        fprintf(stderr, "ERROR: Could not chdir to /: %s\n", strerror(errno));
        return(-1);
    }

    close(STDIN_FILENO);
    close(STDOUT_FILENO);
    close(STDERR_FILENO);

    while ( fgets(line, 256, comm) ) {
        if ( strcmp(line, "stop") == 0 ) {
            printf("Stopping daemon\n");
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

    if ( ( test_daemon_fp = fopen(joinpath(tmpdir, "daemon.pid"), "r") ) == NULL ) {
        fprintf(stderr, "ERROR: Could not open daemon pid file %s: %s\n", joinpath(tmpdir, "daemon.pid"), strerror(errno));
        return(-1);
    }

    daemon_fd = fileno(test_daemon_fp);
    if ( flock(daemon_fd, LOCK_SH | LOCK_NB) == 0 ) {
        fprintf(stderr, "No active container daemon active\n");
        return(0);
    }

    if ( is_fifo(joinpath(tmpdir, "daemon.comm")) < 0 ) {
        fprintf(stderr, "ERROR: Container daemon COMM not available\n");
        return(-1);
    }

    if ( ( comm = fopen(joinpath(tmpdir, "daemon.comm"), "w") ) == NULL ) {
        fprintf(stderr, "Could not open fifo for writing %s: %s\n", joinpath(tmpdir, "daemon.comm"), strerror(errno));
        return(-1);
    }

    fputs("stop", comm);

    fclose(comm);

    return(0);
}



