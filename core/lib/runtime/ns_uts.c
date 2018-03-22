/* 
 * Copyright (c) 2017-2018, SyLabs, Inc. All rights reserved.
 * Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
 * 
 * This software is licensed under a 3-clause BSD license.  Please
 * consult LICENSE file distributed with the sources of this project regarding
 * your rights to use or distribute this software.
 * 
 */

#define _GNU_SOURCE
#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <string.h>
#include <limits.h>
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
#include "util/daemon.h"
#include "util/setns.h"


int _singularity_runtime_ns_uts(void) {
    char *hostname = singularity_registry_get("HOSTNAME");
    size_t hostname_len = 0;

    if ( hostname ) {
        hostname_len = strlen(hostname);
        if ( hostname_len > HOST_NAME_MAX ) {
            singularity_message(WARNING, "Hostname too long, truncated to %d bytes length\n", HOST_NAME_MAX);
            hostname_len = HOST_NAME_MAX;
        }
    }

    if ( singularity_registry_get("UNSHARE_UTS") == NULL ) {
        /* UTS namespace is enforced for root user */
        if ( singularity_priv_getuid() != 0 ) {
            singularity_message(VERBOSE2, "Not virtualizing UTS namespace on user request\n");
            return(0);
        }
    }

#ifdef NS_CLONE_NEWUTS
    singularity_message(DEBUG, "Using UTS namespace: CLONE_NEWUTS\n");
    singularity_priv_escalate();
    singularity_message(DEBUG, "Virtualizing UTS namespace\n");
    if ( unshare(CLONE_NEWUTS) < 0 ) {
        singularity_message(ERROR, "Could not virtualize UTS namespace: %s\n", strerror(errno));
        ABORT(255);
    }
    if ( hostname && sethostname(hostname, hostname_len) < 0 ) {
        singularity_message(ERROR, "Could not set hostname %s: %s\n", hostname, strerror(errno));
        ABORT(255);
    }
    singularity_priv_drop();

#else
    singularity_message(WARNING, "Skipping UTS namespace creation, support not available on host\n");
    return(0);
#endif

    return(0);
}

int _singularity_runtime_ns_uts_join(int ns_fd) {
    int uts_fd;

    /* Attempt to open /proc/[PID]/ns/pid */
    singularity_priv_escalate();
    if ( ! singularity_daemon_own_namespace("uts") ) {
        singularity_priv_drop();
        return(0);
    }
    uts_fd = openat(ns_fd, "uts", O_RDONLY);

    if( uts_fd == -1 ) {
        /* If no IPC file exists, continue without IPC NS */
        singularity_message(WARNING, "Skipping UTS namespace creation, support not available on host\n");
        return(0);
    }
    
    singularity_message(DEBUG, "Attempting to join UTS namespace\n");
    if ( setns(uts_fd, CLONE_NEWUTS) < 0 ) {
        singularity_message(ERROR, "Could not join UTS namespace: %s\n", strerror(errno));
        ABORT(255);
    }
    singularity_priv_drop();
    singularity_message(DEBUG, "Successfully joined UTS namespace\n");

    close(uts_fd);
    return(0);
}
