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

#include <errno.h>
#include <fcntl.h>
#include <signal.h>
#include <string.h>
#include <unistd.h>
#include <poll.h>
#include <sys/types.h>
#include <sys/wait.h>


#include "lib/message.h"
#include "util/util.h"

int generic_signal_rpipe = -1;
int generic_signal_wpipe = -1;
int sigchld_signal_rpipe = -1;
int sigchld_signal_wpipe = -1;
int watchdog_rpipe = -1;
int watchdog_wpipe = -1;
pid_t child_pid;


void handle_signal(int sig, siginfo_t * _, void * __) {
    char info = (char)sig;
    singularity_message(DEBUG, "Forwarding signal through generic_signal_wpipe\n");
    while (-1 == write(generic_signal_wpipe, &info, 1) && errno == EINTR) {}
}

void handle_sigchld(int sig, siginfo_t *siginfo, void * _) {
    singularity_message(DEBUG, "Checking child pids: %i %i\n", siginfo->si_pid, child_pid);
    if ( siginfo->si_pid == child_pid ) {
        singularity_message(DEBUG, "Forwarding signal through sigchld_signal_wpipe\n");
        char one = '1';
        while (-1 == write(sigchld_signal_wpipe, &one, 1) && errno == EINTR) {}
    }
}


pid_t singularity_fork(void) {
    int pipes[2];

    // From: signal_pre_fork()
    if ( pipe2(pipes, O_CLOEXEC) < 0 ) {
        singularity_message(ERROR, "Failed to create watchdog communication pipes: %s\n", strerror(errno));
        ABORT(255);
    }
    watchdog_rpipe = pipes[0];
    watchdog_wpipe = pipes[1];


    // Fork child
    singularity_message(VERBOSE2, "Forking child process\n");
    child_pid = fork();

    if ( child_pid == 0 ) {
        singularity_message(VERBOSE2, "Hello from child process\n");

        if (watchdog_wpipe != -1) {
            singularity_message(DEBUG, "Closing watchdog write pipe\n");
            close(watchdog_wpipe);
        }
        watchdog_wpipe = -1;

        singularity_message(DEBUG, "Child process is returning control to process thread\n");
        return(0);

    } else if ( child_pid > 0 ) {
        singularity_message(VERBOSE2, "Hello from parent process\n");

        // From: setup_signal_handler()
        sigset_t blocked_mask, old_mask, empty_mask;
        sigfillset(&blocked_mask);
        sigemptyset(&empty_mask);
        sigprocmask(SIG_SETMASK, &blocked_mask, &old_mask);

        struct sigaction action;
        action.sa_sigaction = handle_signal;
        action.sa_flags = SA_SIGINFO|SA_RESTART;
        // All our handlers are signal safe.
        action.sa_mask = empty_mask;

        struct pollfd fds[3];
        int retval;
        int child_ok = 1;


        singularity_message(DEBUG, "Assigning sigaction()s\n");
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

        singularity_message(DEBUG, "Creating generic signal pipes\n");
        if ( -1 == pipe2(pipes, O_CLOEXEC) ) {
            singularity_message(ERROR, "Failed to create communication pipes: %s\n", strerror(errno));
            ABORT(255);
        }
        generic_signal_rpipe = pipes[0];
        generic_signal_wpipe = pipes[1];

        singularity_message(DEBUG, "Creating sigcld signal pipes\n");
        if ( -1 == pipe2(pipes, O_CLOEXEC) ) {
            singularity_message(ERROR, "Failed to create communication pipes: %s\n", strerror(errno));
            ABORT(255);
        }
        sigchld_signal_rpipe = pipes[0];
        sigchld_signal_wpipe = pipes[1];

        sigprocmask(SIG_SETMASK, &old_mask, NULL);

        fds[0].fd = sigchld_signal_rpipe;
        fds[0].events = POLLIN;
        fds[0].revents = 0;
        fds[1].fd = generic_signal_rpipe;
        fds[1].events = POLLIN;
        fds[1].revents = 0;
        fds[2].fd = watchdog_rpipe;
        fds[2].events = POLLIN;
        fds[2].revents = 0;


        do {
            singularity_message(DEBUG, "Waiting on signal from watchdog\n");
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
                while (-1 == (retval = read(generic_signal_rpipe, &signum, 1)) && errno == EINTR) {} // Flawfinder: ignore
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

        singularity_message(DEBUG, "Parent process is exiting\n");

        return(child_pid);

    } else {
        singularity_message(ERROR, "Failed to fork child process\n");
        ABORT(255);
    }
}


void singularity_fork_run(void) {
    int tmpstatus;
    int retval = 0;
    pid_t child;

    if ( ( child = singularity_fork() ) > 0 ) {
        singularity_message(DEBUG, "Waiting on child process\n");
                                
        waitpid(child, &tmpstatus, 0);
        retval = WEXITSTATUS(tmpstatus);
        exit(retval);
    }

    return;
}

int singularity_fork_exec(char **argv) {
    int tmpstatus;
    int retval = 0;
    pid_t child;

    child = singularity_fork();

    if ( child == 0 ) {
        if ( execvp(argv[0], argv) < 0 ) { //Flawfinder: ignore
            singularity_message(ERROR, "Failed to execv(%s, ...)\n", argv[0]);
            ABORT(255);
        }

    } else if ( child > 0 ) {
        singularity_message(DEBUG, "Waiting on child process\n");
                                
        waitpid(child, &tmpstatus, 0);
        retval = WEXITSTATUS(tmpstatus);
    }

    return(retval);
}



