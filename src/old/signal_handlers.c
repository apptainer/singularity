/*
 * Copyright (c) 2016, Brian Bockelman. All rights reserved.
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
#include "signal_handlers.h"

#include <errno.h>
#include <fcntl.h>
#include <signal.h>
#include <string.h>
#include <unistd.h>
#include <poll.h>

#include "message.h"
#include "util.h"

pid_t child_pid = 0;
int generic_signal_rpipe = -1;
int generic_signal_wpipe = -1;
int sigchld_signal_rpipe = -1;
int sigchld_signal_wpipe = -1;
int watchdog_rpipe = -1;
int watchdog_wpipe = -1;

void handle_sigchld(int sig, siginfo_t *siginfo, void * _) {
    if ( siginfo->si_pid == child_pid ) {
        char one = '1';
        while (-1 == write(sigchld_signal_wpipe, &one, 1) && errno == EINTR) {}
    }
}

void handle_signal(int sig, siginfo_t * _, void * __) {
    char info = (char)sig;
    while (-1 == write(generic_signal_wpipe, &info, 1) && errno == EINTR) {}
}

void setup_signal_handler(pid_t pid) {
    sigset_t blocked_mask, old_mask, empty_mask;
    sigfillset(&blocked_mask);
    sigemptyset(&empty_mask);
    sigprocmask(SIG_SETMASK, &blocked_mask, &old_mask);
    child_pid = pid;

    struct sigaction action;
    action.sa_sigaction = handle_signal;
    action.sa_flags = SA_SIGINFO|SA_RESTART;
    // All our handlers are signal safe.
    action.sa_mask = empty_mask;

    if ( -1 == sigaction(SIGINT, &action, NULL) ) {
        singularity_message(ERROR, "Failed to install SIGINT signal handler: %s\n", strerror(errno));
        ABORT(255);
    }
    if ( -1 == sigaction(SIGQUIT, &action, NULL) ) {
        singularity_message(ERROR, "Failed to install SIGQUIT signal handler: %s\n", strerror(errno));
        ABORT(255);
    }
    if ( -1 == sigaction(SIGTERM, &action, NULL) ) {
        singularity_message(ERROR, "Failed to install SIGTERM signal handler: %s\n", strerror(errno));
        ABORT(255);
    }
    if ( -1 == sigaction(SIGHUP, &action, NULL) ) {
        singularity_message(ERROR, "Failed to install SIGHUP signal handler: %s\n", strerror(errno));
        ABORT(255);
    }
    if ( -1 == sigaction(SIGUSR1, &action, NULL) ) {
        singularity_message(ERROR, "Failed to install SIGUSR1 signal handler: %s\n", strerror(errno));
        ABORT(255);
    }
    if ( -1 == sigaction(SIGUSR2, &action, NULL) ) {
        singularity_message(ERROR, "Failed to install SIGUSR2 signal handler: %s\n", strerror(errno));
        ABORT(255);
    }
    action.sa_sigaction = handle_sigchld;
    if ( -1 == sigaction(SIGCHLD, &action, NULL) ) {
        singularity_message(ERROR, "Failed to install SIGCHLD signal handler: %s\n", strerror(errno));
        ABORT(255);
    }

    int pipes[2];
    if ( -1 == pipe2(pipes, O_CLOEXEC) ) {
        singularity_message(ERROR, "Failed to create communication pipes: %s\n", strerror(errno));
        ABORT(255);
    }
    generic_signal_rpipe = pipes[0];
    generic_signal_wpipe = pipes[1];

    if ( -1 == pipe2(pipes, O_CLOEXEC) ) {
        singularity_message(ERROR, "Failed to create communication pipes: %s\n", strerror(errno));
        ABORT(255);
    }
    sigchld_signal_rpipe = pipes[0];
    sigchld_signal_wpipe = pipes[1];

    sigprocmask(SIG_SETMASK, &old_mask, NULL);
}


void signal_pre_fork() {
    int pipes[2];
    if ( -1 == pipe2(pipes, O_CLOEXEC) ) {
        singularity_message(ERROR, "Failed to create communication pipes: %s\n", strerror(errno));
        ABORT(255);
    }
    watchdog_rpipe = pipes[0];
    watchdog_wpipe = pipes[1];
}


void signal_post_parent() {
    singularity_message(DEBUG, "Closing watchdog read pipe, FD: %d\n", watchdog_rpipe);
    if (watchdog_rpipe != -1) {
        close(watchdog_rpipe);
    }
    watchdog_rpipe = -1;
}


void signal_post_child() {
    singularity_message(DEBUG, "Closing watchdog write pipe, FD: %d\n", watchdog_wpipe);
    if (watchdog_wpipe != -1) {
        close(watchdog_wpipe);
    }
    watchdog_wpipe = -1;
}


void blockpid_or_signal() {
    struct pollfd fds[3];
    fds[0].fd = sigchld_signal_rpipe;
    fds[0].events = POLLIN;
    fds[0].revents = 0;
    fds[1].fd = generic_signal_rpipe;
    fds[1].events = POLLIN;
    fds[1].revents = 0;
    fds[2].fd = watchdog_rpipe;
    fds[2].events = POLLIN;
    fds[2].revents = 0;
    int retval;
    int child_ok = 1;
    do {
        while ( -1 == (retval = poll(fds, watchdog_rpipe == -1 ? 2 : 3, -1)) && errno == EINTR ) {}
        if ( -1 == retval ) {
            singularity_message(ERROR, "Failed to wait for file descriptors: %s\n", strerror(errno));
            ABORT(255);
        }
        if (fds[0].revents) {
            child_ok = 0;
        }
        if (fds[1].revents) {
            char signum = SIGKILL;
            while (-1 == (retval = read(generic_signal_rpipe, &signum, 1)) && errno == EINTR) {}
            if (-1 == retval) {
                singularity_message(ERROR, "Failed to read from signal handler pipe: %s\n", strerror(errno));
                ABORT(255);
            }
            kill(child_pid, signum);
        }
        if (watchdog_rpipe != -1 && fds[2].revents) {
            // Parent died.  Immediately kill child.
            kill(child_pid, SIGKILL);
            close(watchdog_rpipe);
            watchdog_rpipe = -1;
        }
    } while ( child_ok );
}
