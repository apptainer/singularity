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
#include "util.h"
#include "file.h"
#include "namespaces/ns_mnt.h"


int ns_mnt_init(void) {
    // Function to initalize and check sanity

    // Return zero on success
    return(0);
}

void ns_mnt_unshare(void) {
#ifdef NS_CLONE_FS
    // Setup FS namespaces
    message(DEBUG, "Virtualizing FS namespace\n");
    if ( unshare(CLONE_FS) < 0 ) {
        message(ERROR, "Could not virtualize file system namespace: %s\n", strerror(errno));
        ABORT(255);
    }
#endif

    // Setup mount namespaces
    message(DEBUG, "Virtualizing mount namespace\n");
    if ( unshare(CLONE_NEWNS) < 0 ) {
        message(ERROR, "Could not virtualize mount namespace: %s\n", strerror(errno));
        ABORT(255);
    }
}


void ns_mnt_join(pid_t daemon_pid) {
#ifdef NO_SETNS
    message(ERROR, "This host does not support joining existing name spaces\n");
    ABORT(1);
#else
    char *nsjoin= (char *)malloc(64);

    snprintf(nsjoin, 64, "/proc/%d/ns/mnt", daemon_pid); // Flawfinder: ignore

    if ( is_file(nsjoin) == 0 ) {
        message(DEBUG, "Connecting to existing mount namespace\n");
        int fd = open(nsjoin, O_RDONLY); // Flawfinder: ignore
        if ( setns(fd, CLONE_NEWNS) < 0 ) {
            message(ERROR, "Could not join existing mount namespace: %s\n", strerror(errno));
            ABORT(255);
        }
        close(fd);

    } else {
        message(ERROR, "Could not identify mount namespace: %s\n", nsjoin);
        ABORT(255);
    }
#endif
}
