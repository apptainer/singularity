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
#include "fork.h"


static int enabled = -1;

int singularity_ns_pid_enabled(void) {
    message(DEBUG, "Checking PID namespace enabled: %d\n", enabled);
    return(enabled);
}

int singularity_ns_pid_unshare(void) {

    config_rewind();
    if ( config_get_key_bool("allow pid ns", 1) <= 0 ) {
        message(VERBOSE2, "Not virtualizing PID namespace by configuration\n");
        return(0);
    }

    if ( getenv("SINGULARITY_UNSHARE_PID") == NULL ) { // Flawfinder: ignore (only checking for existance of envar)
        message(VERBOSE2, "Not virtualizing PID namespace on user request\n");
        return(0);
    }

#ifdef NS_CLONE_NEWPID
    message(DEBUG, "Using PID namespace: CLONE_NEWPID\n");
    singularity_priv_escalate();
    message(DEBUG, "Virtualizing PID namespace\n");
    if ( unshare(CLONE_NEWPID) < 0 ) {
        message(ERROR, "Could not virtualize PID namespace: %s\n", strerror(errno));
        ABORT(255);
    }
    singularity_priv_drop();
    enabled = 0;

#else
#ifdef NS_CLONE_PID
    message(DEBUG, "Using PID namespace: CLONE_PID\n");
    singularity_priv_escalate();
    message(DEBUG, "Virtualizing PID namespace\n");
    if ( unshare(CLONE_NEWPID) < 0 ) {
        message(ERROR, "Could not virtualize PID namespace: %s\n", strerror(errno));
        ABORT(255);
    }
    singularity_priv_drop();
    enabled = 0;

#endif
    message(VERBOSE, "Skipping PID namespace creation, support not available\n");
    return(0);
#endif

    // PID namespace requires a fork to activate!
    singularity_fork_run();

    return(0);
}


