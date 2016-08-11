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
#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <string.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/mount.h>
#include <sys/wait.h>
#include <unistd.h>
#include <stdlib.h>
#include <sched.h>


#include "file.h"
#include "util.h"
#include "message.h"
#include "config_parser.h"
#include "privilege.h"


int singularity_ns_pid_unshare(void) {
    pid_t child_ns_pid = 0;
    int retval;

    config_rewind();
#ifdef NS_CLONE_NEWPID
    message(DEBUG, "Using PID namespace: CLONE_NEWPID\n");
    if ( ( getenv("SINGULARITY_NO_NAMESPACE_PID") == NULL ) && // Flawfinder: ignore (only checking for existance of envar)
            ( config_get_key_bool("allow pid ns", 1) > 0 ) ) {
        unsetenv("SINGULARITY_NO_NAMESPACE_PID");
        priv_escalate();
        message(DEBUG, "Virtualizing PID namespace\n");
        if ( unshare(CLONE_NEWPID) < 0 ) {
            message(ERROR, "Could not virtualize PID namespace: %s\n", strerror(errno));
            ABORT(255);
        }
        priv_drop();

        // PID namespace requires a fork to activate!
        child_ns_pid = fork();
    } else {
        message(VERBOSE, "Not virtualizing PID namespace\n");
    }
#else
#ifdef NS_CLONE_PID
    message(DEBUG, "Using PID namespace: CLONE_PID\n");
    if ( ( getenv("SINGULARITY_NO_NAMESPACE_PID") == NULL ) && // Flawfinder: ignore (only checking for existance of envar)
            ( config_get_key_bool("allow pid ns", 1) > 0 ) ) {
        unsetenv("SINGULARITY_NO_NAMESPACE_PID");
        priv_escalate();
        message(DEBUG, "Virtualizing PID namespace\n");
        if ( unshare(CLONE_NEWPID) < 0 ) {
            message(ERROR, "Could not virtualize PID namespace: %s\n", strerror(errno));
            ABORT(255);
        }
        priv_drop();

        // PID namespace requires a fork to activate!
        child_ns_pid = fork();
    } else {
        message(VERBOSE, "Not virtualizing PID namespace\n");
    }
#endif
    message(VERBOSE, "Skipping PID namespace creation, support not available\n");
    return(0);
#endif

    if ( child_ns_pid == 0 ) {
        // Allow the child to continue on, while we catch the parent...
    } else if ( child_ns_pid > 0 ) {
        int tmpstatus;

        message(DEBUG, "Waiting on NS child process\n");

        waitpid(child_ns_pid, &tmpstatus, 0);
        retval = WEXITSTATUS(tmpstatus);
        exit(retval);
    } else {
        message(ERROR, "Failed forking child process\n");
        ABORT(255);
    }

    return(0);
}


