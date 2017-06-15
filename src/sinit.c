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
    int i, host_pid;
    int *daemon_file_fd = malloc(sizeof(int));
    ssize_t bufsize = 2048;
    char *daemon_file;
    char *host_pid_str = malloc(bufsize);
    
    singularity_config_init(joinpath(SYSCONFDIR, "/singularity/singularity.conf"));
    singularity_registry_init();
    daemon_path(argv[1]);

    /* Fork into sinit daemon inside PID NS */
    singularity_fork_daemonize();

    /* After this point, we are running as PID 1 inside PID NS */
    singularity_message(DEBUG, "Preparing sinit daemon\n");

    /* Calling readlink on /proc/self returns the PID of the thread in the host PID NS */
    if ( readlink("/proc/self", host_pid_str, bufsize) == -1 ) {
        singularity_message(ERROR, "Unable to open /proc/self: %s\n", strerror(errno));
        ABORT(255);
    } else {
        singularity_message(DEBUG, "PID in host namespace, from /proc/self: %s\n", host_pid_str);
        host_pid = atoi(host_pid_str);
    }

    /* Get pathname of daemon information file */
    daemon_file = singularity_registry_get("DAEMON_FILE");

    /* Check if /tmp/.singularity-daemon-[UID]/ directory exists, if not create it */
    if( is_dir(dirname(singularity_registry_get("DAEMON_FILE"))) == -1 )
        s_mkpath(dirname(singularity_registry_get("DAEMON_FILE")), 0755);

    /* Attempt to open lock on daemon file */
    i = filelock(daemon_file, daemon_file_fd);

    if( i == 0 ) {
        singularity_message(DEBUG, "Successfully obtained excluse lock on %s\n", daemon_file);
        
        /* Successfully obtained lock, write [PID] to open fd */
        if( write(*daemon_file_fd, host_pid_str, strlength(host_pid_str, 2048) + 1) == -1 ) {
            singularity_message(ERROR, "Unable to write to %s: %s\n", daemon_file, strerror(errno));
        }
    } else if( i == EALREADY ) {
        /* Another daemon controls this file already */
        singularity_message(ERROR, "Daemon already exists: %s\n", strerror(errno));
        ABORT(255);
    } else {
        singularity_message(ERROR, "Cannot lock %s: %s\n", daemon_file, strerror(errno));
        ABORT(255);
    }
    
    /* Close all open fd's that may be present besides daemon info file fd */
    singularity_message(DEBUG, "Closing open fd's\n");
    for( i = sysconf(_SC_OPEN_MAX); i >= 0; i-- ) {
        if( i != *daemon_file_fd )
            close(i);
    }
    
    singularity_message(LOG, "Successfully closed fd's, entering daemon loop\n");

    i = 0;
    while(1) {
        if( i++ == 2 )
            break;
        singularity_message(LOG, "Logging from inside daemon\n");
        sleep(60);
    }
    
    return(0);
}
