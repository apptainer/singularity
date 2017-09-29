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
#include <signal.h>
#include <poll.h>

#include "config.h"
#include "util/file.h"
#include "util/fork.h"
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


static struct pollfd fds[1];
static int pipes[2] = {-1, -1};


void sigchld_sinit(int sig, siginfo_t *siginf, void *_);
static void install_sigchld_handle();
static void wait_procs();


int main(int argc, char **argv) {
    if (chdir("/") < 0 ) {
        singularity_message(ERROR, "Can't change directory to /\n");
    }
    setsid();
    umask(0);

    install_sigchld_handle();
    wait_procs();
    
    return(0);
}

/* Sigaction function, write PID of process that sent SIGCHLD to signal pipe */
void sigchld_sinit(int sig, siginfo_t *siginf, void *_) {
    pid_t child_pid = siginf->si_pid;
    while( -1 == write(pipes[1], &child_pid, sizeof(pid_t)) && errno == EINTR ) {}
}

/* Will poll on signal pipe and when SIGCHLD is received call waitpid(child_pid) */
static void wait_procs() {
    pid_t child_pid;
    int retval, tmpstatus;

    while(1) {
        /* Poll on fds for a POLLIN event */
        while ( -1 == (retval = poll(fds, 1, -1)) && errno == EINTR ) {}

        if ( -1 == retval ) {
            singularity_message(LOG, "Failed to wait for fds: %s\n", strerror(errno));
            ABORT(255);
        }

        if ( fds[0].revents ) {
            while (-1 == (retval = read(fds[0].fd, &child_pid, sizeof(pid_t))) && errno == EINTR) {} // Flawfinder: ignore

            if ( retval == -1 ) {
                singularity_message(LOG, "Failed to read from signal handler pipe: %s\n", strerror(errno));
                ABORT(255);
            }
            
            singularity_message(LOG, "SIGCHILD raised from child: %d\n", child_pid);
            waitpid(child_pid, &tmpstatus, 0);
        }

    }
}


/* Set sigchld signal handler */
static void install_sigchld_handle() {
    struct sigaction action;
    sigset_t empty_mask;
    
    sigemptyset(&empty_mask);

    /* Fill action with handle_sigchld function */
    action.sa_sigaction = &sigchld_sinit;
    action.sa_flags = SA_SIGINFO|SA_RESTART;
    action.sa_mask = empty_mask;
    
    singularity_message(DEBUG, "Assigning SIGCHLD sigaction()\n");
    if ( -1 == sigaction(SIGCHLD, &action, NULL) ) {
        singularity_message(ERROR, "Failed to install SIGCHLD signal handler: %s\n", strerror(errno));
        ABORT(255);
    }
    
    /* Open pipes for handle_sigchld() to write to */
    singularity_message(DEBUG, "Creating sigchld signal pipes\n");
    if ( -1 == pipe2(pipes, O_CLOEXEC) ) {
        singularity_message(ERROR, "Failed to create communication pipes: %s\n", strerror(errno));
        ABORT(255);
    }

    /* Fill fds struct with read pipe */
    fds[0].fd = pipes[0];
    fds[0].events = POLLIN;
    fds[0].revents = 0;
}
