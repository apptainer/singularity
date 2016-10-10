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


#include "util/file.h"
#include "util/util.h"
#include "lib/message.h"
#include "lib/config_parser.h"
#include "lib/singularity.h"


int singularity_ns_unshare(void) {
    int retval = 0;

    singularity_message(DEBUG, "Unsharing all namespaces\n");
    retval += singularity_ns_user_unshare();
    retval += singularity_ns_pid_unshare();
    retval += singularity_ns_mnt_unshare();
    retval += singularity_ns_ipc_unshare();

    return(retval);
}


int singularity_ns_join(pid_t attach_pid) {
#ifdef NO_SETNS
    singularity_message(ERROR, "This host does not support joining existing name spaces\n");
    ABORT(1);
#else
    char *nsjoin_pid = (char *)malloc(64);
    char *nsjoin_mnt = (char *)malloc(64);
    char *nsjoin_ipc = (char *)malloc(64);

    snprintf(nsjoin_pid, 64, "/proc/%d/ns/pid", attach_pid); // Flawfinder: ignore
    snprintf(nsjoin_mnt, 64, "/proc/%d/ns/mnt", attach_pid); // Flawfinder: ignore
    snprintf(nsjoin_ipc, 64, "/proc/%d/ns/ipc", attach_pid); // Flawfinder: ignore

    if ( is_file(nsjoin_pid) == 0 ) {
        singularity_message(DEBUG, "Connecting to existing PID namespace\n");
        int fd = open(nsjoin_pid, O_RDONLY); // Flawfinder: ignore
        if ( setns(fd, CLONE_NEWPID) < 0 ) {
            singularity_message(ERROR, "Could not join existing PID namespace: %s\n", strerror(errno));
            ABORT(255);
        }
        close(fd);

    } else {
        singularity_message(ERROR, "Could not identify PID namespace: %s\n", nsjoin_pid);
        ABORT(255);
    }

    if ( is_file(nsjoin_mnt) == 0 ) {
        singularity_message(DEBUG, "Connecting to existing mount namespace\n");
        int fd = open(nsjoin_mnt, O_RDONLY); // Flawfinder: ignore
        if ( setns(fd, CLONE_NEWNS) < 0 ) {
            singularity_message(ERROR, "Could not join existing mount namespace: %s\n", strerror(errno));
            ABORT(255);
        }
        close(fd);

    } else {
        singularity_message(ERROR, "Could not identify mount namespace: %s\n", nsjoin_mnt);
        ABORT(255);
    }

    if ( is_file(nsjoin_ipc) == 0 ) {
        singularity_message(DEBUG, "Connecting to existing IPC namespace\n");
        int fd = open(nsjoin_ipc, O_RDONLY); // Flawfinder: ignore
        if ( setns(fd, CLONE_NEWIPC) < 0 ) {
            singularity_message(ERROR, "Could not join existing IPC namespace: %s\n", strerror(errno));
            ABORT(255);
        }
        close(fd);

    } else {
        singularity_message(ERROR, "Could not identify IPC namespace: %s\n", nsjoin_ipc);
        ABORT(255);
    }
#endif
    return(0);
}

