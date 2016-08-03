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
#include <unistd.h>
#include <stdlib.h>
#include <sched.h>

#include "message.h"
#include "config_parser.h"
#include "util.h"
#include "file.h"
#include "namespaces/ns_pid.h"


int ns_pid_init(void) {
    // Function to initalize and check sanity

    // Return zero on success
    return(0);
}


void ns_pid_unshare(void) {
    config_rewind();
#ifdef NS_CLONE_NEWPID
    if ( ( getenv("SINGULARITY_NO_NAMESPACE_PID") == NULL ) && // Flawfinder: ignore (only checking for existance of envar)
            ( config_get_key_bool("allow pid ns", 1) > 0 ) ) {
        unsetenv("SINGULARITY_NO_NAMESPACE_PID");
        message(DEBUG, "Virtualizing PID namespace\n");
        if ( unshare(CLONE_NEWPID) < 0 ) {
            message(ERROR, "Could not virtualize PID namespace: %s\n", strerror(errno));
            ABORT(255);
        }
    } else {
        message(VERBOSE, "Not virtualizing PID namespace\n");
    }
#else
#ifdef NS_CLONE_PID
    if ( ( getenv("SINGULARITY_NO_NAMESPACE_PID") == NULL ) && // Flawfinder: ignore (only checking for existance of envar)
            ( config_get_key_bool("allow pid ns", 1) > 0 ) ) {
        unsetenv("SINGULARITY_NO_NAMESPACE_PID");
        message(DEBUG, "Virtualizing PID namespace\n");
        if ( unshare(CLONE_NEWPID) < 0 ) {
            message(ERROR, "Could not virtualize PID namespace: %s\n", strerror(errno));
            ABORT(255);
        }
    } else {
        message(VERBOSE, "Not virtualizing PID namespace\n");
    }
#endif
#endif
}


void ns_pid_join(pid_t daemon_pid) {
#ifdef NO_SETNS
    message(ERROR, "This host does not support joining existing name spaces\n");
    ABORT(1);
#else
    char *nsjoin= (char *)malloc(64);

    snprintf(nsjoin, 64, "/proc/%d/ns/pid", daemon_pid); // Flawfinder: ignore

    if ( is_file(nsjoin) == 0 ) {
        message(DEBUG, "Connecting to existing PID namespace\n");
        int fd = open(nsjoin, O_RDONLY); // Flawfinder: ignore
        if ( setns(fd, CLONE_NEWPID) < 0 ) {
            message(ERROR, "Could not join existing PID namespace: %s\n", strerror(errno));
            ABORT(255);
        }
        close(fd);

    } else {
        message(ERROR, "Could not identify PID namespace: %s\n", nsjoin);
        ABORT(255);
    }
#endif
}


