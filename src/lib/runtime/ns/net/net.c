/* 
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
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/mount.h>
#include <sys/wait.h>
#include <sys/ioctl.h>
#include <net/if.h>
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


static int enabled = -1;

int _singularity_runtime_ns_net(void) {
    int sockfd;
    struct ifreq req;
    
    if ( singularity_registry_get("UNSHARE_NET") == NULL ) {
        singularity_message(VERBOSE2, "Not virtualizing network namespace on user request\n");
        return(0);
    }

#ifdef NS_CLONE_NEWNET
    singularity_message(DEBUG, "Using network namespace: CLONE_NEWNET\n");
    singularity_priv_escalate();
    singularity_message(DEBUG, "Virtualizing network namespace\n");
    if ( unshare(CLONE_NEWNET) < 0 ) {
        singularity_message(ERROR, "Could not virtualize network namespace: %s\n", strerror(errno));
        ABORT(255);
    }
    singularity_priv_drop();
    enabled = 0;

#else
    singularity_message(WARNING, "Skipping network namespace creation, support not available on host\n");
    return(0);
#endif

    sockfd = socket(AF_INET, SOCK_DGRAM, 0);

    if ( sockfd < 0 ) {
        singularity_message(ERROR, "Unable to open AF_INET socket: %s\n", strerror(errno));
        ABORT(255);
    }
    
    memset(&req, 0, sizeof(req));
    strncpy(req.ifr_name, "lo", IFNAMSIZ);

    req.ifr_flags |= IFF_UP;

    singularity_priv_escalate();
    singularity_message(DEBUG, "Bringing up network loopback interface\n");
    if ( ioctl(sockfd, SIOCSIFFLAGS, &req) < 0 ) {
        singularity_message(ERROR, "Failed to set flags on interface: %s\n", strerror(errno));
        ABORT(255);
    }
    singularity_priv_drop();
    
    return(0);
}

int _singularity_runtime_ns_net_join(void) {
    int ns_fd = atoi(singularity_registry_get("DAEMON_NS_FD"));
    int net_fd;

    /* Attempt to open /proc/[PID]/ns/net */
    singularity_priv_escalate();
    net_fd = openat(ns_fd, "net", O_RDONLY);

    if( net_fd == -1 ) {
        singularity_message(ERROR, "Could not open NET NS fd: %s\n", strerror(errno));
        ABORT(255);
    }
    
    singularity_message(DEBUG, "Attempting to join NET namespace\n");
    if ( setns(net_fd, CLONE_NEWNET) < 0 ) {
        singularity_message(ERROR, "Could not join NET namespace: %s\n", strerror(errno));
        ABORT(255);
    }
    singularity_priv_drop();
    singularity_message(DEBUG, "Successfully joined NET namespace\n");

    close(net_fd);
    return(0);
}

/*
int singularity_ns_net_enabled(void) {
    singularity_message(DEBUG, "Checking NET namespace enabled: %d\n", enabled);
    return(enabled);
}
*/
