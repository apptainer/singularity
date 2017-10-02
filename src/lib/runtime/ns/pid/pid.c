/* 
 * Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
 *
 * Copyright (c) 2015-2017, Gregory M. Kurtzer. All rights reserved.
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
#include "util/message.h"
#include "util/config_parser.h"
#include "util/privilege.h"
#include "util/fork.h"
#include "util/registry.h"
#include "util/setns.h"


int _singularity_runtime_ns_pid(void) {

#ifdef SINGULARITY_NO_NEW_PRIVS
    // Use PID namespace when NO_NEW_PRIVS is not supported
    if ( singularity_config_get_bool(ALLOW_PID_NS) <= 0 ) {
        singularity_message(VERBOSE2, "Not virtualizing PID namespace by configuration\n");
        return(0);
    }

    if ( singularity_registry_get("UNSHARE_PID") == NULL ) {
        singularity_message(VERBOSE2, "Not virtualizing PID namespace on user request\n");
        return(0);
    }
#endif /* SINGULARITY_NO_NEW_PRIVS */

#ifdef NS_CLONE_NEWPID
    singularity_message(DEBUG, "Using PID namespace: CLONE_NEWPID\n");
    
#else
    singularity_message(WARNING, "Skipping PID namespace creation, support not available on host\n");
    return(0);
#endif

    singularity_message(DEBUG, "Virtualizing PID namespace\n");
        
    if ( singularity_registry_get("DAEMON_START") ) {
        singularity_fork_daemonize(CLONE_NEWPID);
    } else {
        singularity_fork_run(CLONE_NEWPID);
    }

    singularity_registry_set("PIDNS_ENABLED", "1");

    return(0);
}

int _singularity_runtime_ns_pid_join(void) {
    int ns_fd = atoi(singularity_registry_get("DAEMON_NS_FD"));
    int pid_fd;

    /* Attempt to open /proc/[PID]/ns/pid */
    singularity_priv_escalate();
    pid_fd = openat(ns_fd, "pid", O_RDONLY);

    if( pid_fd == -1 ) {
        /* Daemons should always have a ns/pid file. If it doesn't exist, something is wrong */
        singularity_message(ERROR, "Could not open PID NS fd: %s\n", strerror(errno));
        ABORT(255);
    }
    
    singularity_message(DEBUG, "Attempting to join PID namespace\n");
    if ( setns(pid_fd, CLONE_NEWPID) < 0 ) {
        singularity_message(ERROR, "Could not join PID namespace: %s\n", strerror(errno));
        ABORT(255);
    }
    singularity_priv_drop();
    singularity_message(DEBUG, "Successfully joined PID namespace\n");
    
    close(pid_fd);
    
    /* Enable PID NS by forking into a child */
    singularity_fork_run(0);
    singularity_registry_set("PIDNS_ENABLED", "1");
    
    return(0);
}
