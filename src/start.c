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
#include <sys/wait.h>
#include <sys/prctl.h>
#include <signal.h>
#include <poll.h>

#include "config.h"
#include "util/file.h"
#include "util/util.h"
#include "util/registry.h"
#include "util/fork.h"
#include "lib/image/image.h"
#include "lib/runtime/runtime.h"
#include "util/config_parser.h"
#include "util/privilege.h"
#include "util/suid.h"
#include "util/sessiondir.h"
#include "util/cleanupd.h"
#include "util/daemon.h"
#include "util/signal.h"

#include "./action-lib/include.h"

#ifndef SYSCONFDIR
#error SYSCONFDIR not defined
#endif

/* remove daemon pid file if instance start failed */
static void clean_exit(int code) {
    if ( code != 0 ) {
        char *daemon_file = singularity_registry_get("DAEMON_FILE");
        if ( unlink(daemon_file) < 0 ) {
            singularity_message(ERROR, "Failed to delete %s\n", daemon_file);
        }
    }
    ABORT(code);
}

int main(int argc, char **argv) {
    struct image_object image;
    char *daemon_fd_str;
    int daemon_fd, sig_fd, i;
    pid_t child;

    singularity_config_init(joinpath(SYSCONFDIR, "/singularity/singularity.conf"));

    singularity_priv_init();
    singularity_suid_init(argv);

    singularity_registry_init();
    singularity_priv_userns();
    singularity_priv_drop();

    singularity_registry_set("UNSHARE_PID", "1");
    singularity_registry_set("UNSHARE_IPC", "1");

    singularity_cleanupd();

    if ( singularity_registry_get("WRITABLE") != NULL ) {
        singularity_message(VERBOSE3, "Instantiating writable container image object\n");
        image = singularity_image_init(singularity_registry_get("IMAGE"), O_RDWR);
    } else {
        singularity_message(VERBOSE3, "Instantiating read only container image object\n");
        image = singularity_image_init(singularity_registry_get("IMAGE"), O_RDONLY);
    }
        
    singularity_runtime_ns(SR_NS_ALL);
    
    singularity_sessiondir();

    singularity_image_mount(&image, CONTAINER_MOUNTDIR);

    action_ready();

    singularity_runtime_overlayfs();
    singularity_runtime_mounts();
    singularity_runtime_files();

    /* After this point, we are running as PID 1 inside PID NS */
    singularity_message(DEBUG, "Preparing sinit daemon\n");
    singularity_registry_set("ROOTFS", CONTAINER_FINALDIR);
    singularity_daemon_init();

    singularity_message(DEBUG, "We are ready to recieve jobs, sending signal_go_ahead to parent\n");

    daemon_fd_str = singularity_registry_get("DAEMON_FD");
    daemon_fd = atoi(daemon_fd_str);

    singularity_message(DEBUG, "Signaling parent it is ok to go ahead\n");
    singularity_signal_go_ahead(0);

    singularity_runtime_enter();
    singularity_priv_drop_perm();

    sig_fd = singularity_install_signal_fd();

    singularity_message(DEBUG, "Exited sigfd\n");

    /* Close all open fd's that may be present besides daemon info file fd */
    singularity_message(DEBUG, "Closing open fd's\n");
    for( i = sysconf(_SC_OPEN_MAX); i >= 2; i-- ) {
        if( i != daemon_fd && i != sig_fd) {
            close(i);
        }
    }
    
    if ( chdir("/") < 0 ) {
        singularity_message(ERROR, "Can't change directory to /\n");
        clean_exit(255);
    }
    setsid();
    umask(0);

    child = fork();
    
    if ( child == 0 ) {
        singularity_unblock_signals();
        
        if ( is_exec("/.singularity.d/actions/start") == 0 ) {
            singularity_message(DEBUG, "Exec'ing /.singularity.d/actions/start\n");
            
            if ( execv("/.singularity.d/actions/start", argv) < 0 ) { // Flawfinder: ignore
                singularity_message(ERROR, "Failed to execv() /.singularity.d/actions/start: %s\n", strerror(errno));
            }
        }
        singularity_message(WARNING, "Start script not found\n");
        exit(0);
    } else if ( child > 0 ) {
        singularity_handle_signals(sig_fd);
    } else {
        clean_exit(255);
    }
    clean_exit(0);
}
