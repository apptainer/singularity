/* 
 * Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
 * 
 * This software is licensed under a 3-clause BSD license.  Please
 * consult LICENSE file distributed with the sources of this project regarding
 * your rights to use or distribute this software.
 * 
 */

#define _GNU_SOURCE
#include <sys/signalfd.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <stdio.h>
#include <signal.h>
#include <unistd.h>
#include <errno.h>
#include <string.h>

#include "config.h"
#include "util/util.h"
#include "util/signal.h"

static int sigfd = -1;
static sigset_t old_mask;

static const int all_signals[] = {
    SIGHUP,
    SIGINT,
    SIGQUIT,
    SIGTRAP,
    SIGIOT,
    SIGUSR1,
    SIGUSR2,
    SIGPIPE,
    SIGALRM,
    SIGTERM,
    SIGSTKFLT,
    SIGCHLD,
    SIGCONT,
    SIGTSTP,
    SIGTTIN,
    SIGTTOU,
    SIGURG,
    SIGXCPU,
    SIGXFSZ,
    SIGVTALRM,
    SIGPROF,
    SIGWINCH,
    SIGIO,
    SIGPOLL,
    SIGPWR,
    SIGSYS
};

static void handle_sig_sigchld(struct signalfd_siginfo *siginfo) {
    if ( waitpid(siginfo->ssi_pid, NULL, WNOHANG) <= 0 ) {
        singularity_message(ERROR, "Unable to wait on child: %s\n", strerror(errno));
    }
}

static void handle_sig_generic(struct signalfd_siginfo *siginfo) {
    singularity_message(DEBUG, "Generic sig received: %d\n", siginfo->ssi_signo);
    kill(-1,  siginfo->ssi_signo);
}

int singularity_install_signal_fd() {
    sigset_t sig_mask;
    int i = 0;

    singularity_message(DEBUG, "Creating signalfd to handle signals\n");
    
    sigemptyset(&sig_mask);
    while( all_signals[i] != 0 ) {
        sigaddset(&sig_mask, all_signals[i]);
        ++i;
    }

    if ( -1 == sigprocmask(SIG_SETMASK, &sig_mask, &old_mask) ) {
        singularity_message(ERROR, "Unable to block signals: %s\n", strerror(errno));
        ABORT(255);
    }

    if ( -1 == (sigfd = signalfd(-1, &sig_mask, SFD_CLOEXEC)) ) {
        singularity_message(ERROR, "Unable to open signalfd: %s\n", strerror(errno));
        ABORT(255);
    }

    return(sigfd);
}

/* Never returns. Will always read from sig_fd and wait for signal events */
void singularity_handle_signals(int sig_fd) {
    ssize_t size = sizeof(struct signalfd_siginfo);
    struct signalfd_siginfo *siginfo = (struct signalfd_siginfo *)malloc(size);

    singularity_message(DEBUG, "Waiting for signals\n");
    
    while(1) {
        if ( read(sig_fd, siginfo, size) != size ) {
            singularity_message(ERROR, "Unable to read sfd: %s\n", strerror(errno));
            ABORT(255);
        }

        if ( siginfo->ssi_signo == SIGCHLD ) {
            handle_sig_sigchld(siginfo);
        } else {
            handle_sig_generic(siginfo);
        }

        memset(siginfo, 0, size);
    }
}

void singularity_unblock_signals() {
    sigprocmask(SIG_SETMASK, &old_mask, NULL);
}
