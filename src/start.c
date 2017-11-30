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

#define CHILD_FAILED    200

int started = 0;

int main(int argc, char **argv) {
    int i, daemon_fd, cleanupd_fd;
    struct tempfile *stdout_log, *stderr_log, *singularity_debug;
    struct image_object image;
    pid_t child;
    siginfo_t siginfo;
    struct stat filestat;

    singularity_config_init(joinpath(SYSCONFDIR, "/singularity/singularity.conf"));

    singularity_priv_init();
    singularity_suid_init(argv);

    singularity_registry_init();
    singularity_priv_userns();
    singularity_priv_drop();

    singularity_runtime_autofs();

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
    
    singularity_runtime_enter();
    singularity_priv_drop_perm();

    if ( envclean() != 0 ) {
        singularity_message(ERROR, "Failed sanitizing the environment\n");
        ABORT(255);
    }

    singularity_install_signal_handler();

    daemon_fd = atoi(singularity_registry_get("DAEMON_FD"));
    cleanupd_fd = atoi(singularity_registry_get("CLEANUPD_FD"));
    
    /* Close all open fd's that may be present besides daemon info file fd */
    singularity_message(DEBUG, "Closing open fd's\n");
    for( i = sysconf(_SC_OPEN_MAX); i > 2; i-- ) {        
        if ( i != daemon_fd && i != cleanupd_fd ) {
            if ( fstat(i, &filestat) == 0 ) {
                if ( S_ISFIFO(filestat.st_mode) != 0 ) {
                    continue;
                }
            }
            close(i);
        }
    }

    singularity_debug = make_logfile("singularity-debug");
    stdout_log = make_logfile("stdout");
    stderr_log = make_logfile("stderr");
    
    for( i = 0; i <= 2; i++ ) {
        close(i);
    }

    if ( chdir("/") < 0 ) {
        singularity_message(ERROR, "Can't change directory to /\n");
    }
    setsid();
    umask(0);

    /* set program name */
    if ( prctl(PR_SET_NAME, "sinit", 0, 0, 0) < 0 ) {
        singularity_message(ERROR, "Failed to set program name\n");
        ABORT(255);
    }

    child = fork();
    
    if ( child == 0 ) {
        /* Make standard output and standard error files to log stdout & stderr into */
        if ( stdout_log != NULL ) {
            if ( -1 == dup2(stdout_log->fd, 1) ) {
                singularity_message(ERROR, "Unable to dup2(): %s\n", strerror(errno));
                ABORT(255);
            }
        }

        if ( stderr_log != NULL ) {
            if ( -1 == dup2(stderr_log->fd, 2) ) {
                singularity_message(ERROR, "Unable to dup2(): %s\n", strerror(errno));
                ABORT(255);
            }
        }

        /* Unblock signals and execute startscript */
        singularity_unblock_signals();
        if ( is_exec("/.singularity.d/actions/start") == 0 ) {
            singularity_message(DEBUG, "Exec'ing /.singularity.d/actions/start\n");

            if ( execv("/.singularity.d/actions/start", argv) < 0 ) { // Flawfinder: ignore
                singularity_message(ERROR, "Failed to execv() /.singularity.d/actions/start: %s\n", strerror(errno));
                ABORT(CHILD_FAILED);
            }
        } else {
            singularity_message(VERBOSE, "Instance start script not found\n");
            kill(1, SIGCONT);
        }
    } else if ( child > 0 ) {
        if ( singularity_debug != NULL ) {
            if ( -1 == dup2(singularity_debug->fd, 2) ) {
                singularity_message(ERROR, "Unable to dup2(): %s\n", strerror(errno));
                ABORT(255);
            }
        }

        singularity_message(DEBUG, "Waiting for signals\n");
        /* send a SIGALRM if start script doesn't send SIGCONT within 1 seconds */
        alarm(1);
        while (1) {
            if ( singularity_handle_signals(&siginfo) < 0 ) {
                singularity_signal_go_ahead(255);
                break;
            }
            if ( siginfo.si_signo == SIGCHLD ) {
                singularity_message(DEBUG, "Child exited\n");
                if ( siginfo.si_pid == 2 && siginfo.si_status == CHILD_FAILED ) {
                    singularity_signal_go_ahead(CHILD_FAILED);
                    break;
                }
            } else if ( siginfo.si_signo == SIGCONT && siginfo.si_pid == 2 ) {
                /* start script correctly exec */
                singularity_signal_go_ahead(0);
                started = 1;
            } else if ( siginfo.si_signo == SIGALRM && started == 0 ) {
                /* don't receive SIGCONT, start script modified/replaced ? */
                singularity_message(ERROR, "Start script doesn't send SIGCONT\n");
                singularity_signal_go_ahead(255);
                break;
            }
        }
    } else {
        singularity_message(ERROR, "Failed to execute start script\n");
        singularity_signal_go_ahead(255);
    }
    return(0);
}
