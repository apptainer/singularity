/* 
 * Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
 * 
 * This software is licensed under a 3-clause BSD license.  Please
 * consult LICENSE file distributed with the sources of this project regarding
 * your rights to use or distribute this software.
 * 
 */


#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <errno.h>
#include <string.h>
#include <fcntl.h>
#include <libgen.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <sys/un.h>

#include "config.h"
#include "util/file.h"
#include "util/util.h"
#include "util/daemon.h"
#include "util/registry.h"
#include "lib/image/image.h"
#include "lib/runtime/runtime.h"
#include "util/config_parser.h"
#include "util/fork.h"
#include "util/privilege.h"
#include "util/suid.h"
#include "util/sessiondir.h"
#include "util/cleanupd.h"

#include "./action-lib/include.h"

#ifndef SYSCONFDIR
#error SYSCONFDIR not defined
#endif


int main(int argc, char **argv) {
    char *daemon_fd_str;
    int daemon_fd;
    int i;
    
    singularity_config_init(joinpath(SYSCONFDIR, "/singularity/singularity.conf"));
    singularity_priv_init();
    singularity_registry_init();

    /* After this point, we are running as PID 1 inside PID NS */
    singularity_message(DEBUG, "Preparing sinit daemon\n");
    singularity_registry_set("ROOTFS", argv[1]);
    singularity_daemon_init();

    daemon_fd_str = singularity_registry_get("DAEMON_FD");
    daemon_fd = atoi(daemon_fd_str);

    /* Close all open fd's that may be present besides daemon info file fd */
    singularity_message(DEBUG, "Closing open fd's\n");
    for( i = sysconf(_SC_OPEN_MAX); i >= 0; i-- ) {
        if( i != daemon_fd ) {
            close(i);
        }
    }
    
    singularity_message(LOG, "Successfully closed fd's, entering daemon loop\n");

    while(1) {
        //singularity_message(LOG, "Logging from inside daemon\n");
        sleep(60);
    }
    
    return(0);
}
