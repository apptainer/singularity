/*
 * Copyright (c) 2017-2018, SyLabs, Inc. All rights reserved.
 * Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
 * Copyright (c) 2016, Brian Bockelman. All rights reserved.
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

#include <errno.h>
#include <fcntl.h>
#include <signal.h>
#include <string.h>
#include <setjmp.h>
#include <sched.h>
#include <unistd.h>
#include <poll.h>
#include <sys/types.h>
#include <sys/resource.h>
#include <sys/wait.h>
#include <stdio.h>


#include "util/privilege.h"
#include "util/message.h"
#include "util/suid.h"
#include "util/util.h"

int signal_rpipe = -1;
int signal_wpipe = -1;
int watchdog_rpipe = -1;
int watchdog_wpipe = -1;
pid_t child_pid;

struct pollfd fdstruct;

typedef struct fork_state_s
{
    sigjmp_buf env;
} fork_state_t;


// NOTE: singularity_message is NOT signal handler safe.
// Hence, we MUST NOT do any sort of generic logging from these
// functions.  We might, in the future, add in a signal-safe
// version of singularity_message here.
void handle_signal(int sig, siginfo_t * _, void * __) {
    char info = (char)sig;
    while (-1 == write(signal_wpipe, &info, 1) && errno == EINTR) {}
}


/* Setup the communication between parent and child.
 *
 * These series of functions force the child to wait for an explicit go-ahead from the
 * parent before proceeding.
 * - `prepare_for_fork`: Must be called by both parent and child before fork() is called.
 * - `wait_for_go_ahead`: Called by child, waits until the parent gives the go-ahead signal.
 * - `signal_go_ahead`: Called by parent, indicates the child may proceed.
 *
 * Keeps global state in the coordination_pipe variable.
 */
static int pipe_to_child[] = {-1, -1};
static int pipe_to_parent[] = {-1, -1};
static int coordination_pipe[] = {-1, -1};

static void prepare_fork() {
    singularity_message(DEBUG, "Creating parent/child coordination pipes.\n");
    // Note we use pipe and not pipe2 here with CLOEXEC.  This is because we eventually want the parent process
    // to exec a separate unprivileged process and inherit the communication pipe.
    if ( -1 == pipe(pipe_to_child) ) {
        singularity_message(ERROR, "Failed to create coordination pipe for fork: %s (errno=%d)\n", strerror(errno), errno);
        ABORT(255);
    }

    if ( -1 == pipe(pipe_to_parent) ) {
        singularity_message(ERROR, "Failed to create coordination pipe for fork: %s (errno=%d)\n", strerror(errno), errno);
        ABORT(255);
    }
    
}

/* Put the appropriate read and write pipes into coordination_pipe[] */
static void prepare_pipes_child() {
    /* Close to child write pipe */
    close(pipe_to_child[1]);

    /* Close to parent read pipe */
    close(pipe_to_parent[0]);

    /* Store read and write pipes into common variable */
    coordination_pipe[0] = pipe_to_child[0];
    coordination_pipe[1] = pipe_to_parent[1];
}

/* Put the appropriate read and write pipes into coordination_pipe[] */
static void prepare_pipes_parent() {
    /* Close to parent write pipe */
    close(pipe_to_parent[1]);

    /* Close to child read pipe */
    close(pipe_to_child[0]);

    /* Store read and write pipes into common variable */
    coordination_pipe[0] = pipe_to_parent[0];
    coordination_pipe[1] = pipe_to_child[1];
}

/* Updated wait_for_go_ahead() which allows bi-directional wait signaling */
int singularity_wait_for_go_ahead() {
    if ( (coordination_pipe[0] == -1) || (coordination_pipe[1] == -1)) {
        singularity_message(ERROR, "Internal error!  wait_for_go_ahead invoked with invalid pipe state (%d, %d).\n",
                            coordination_pipe[0], coordination_pipe[1]);
        ABORT(255);
    }

    singularity_message(DEBUG, "Waiting for go-ahead signal\n");
    
    char code = -1;
    int retval;

    // Block until other process indicates it is OK to proceed.
    while ( (-1 == (retval = read(coordination_pipe[0], &code, 1))) && errno == EINTR) {}

    if (retval == -1) {  // Failed to communicate with other process.
        singularity_message(ERROR, "Failed to communicate with other process: %s (errno=%d)\n", strerror(errno), errno);
        ABORT(255);
    } else if (retval == 0) {  // Other process closed the write pipe unexpectedly.
        if ( close(dup(coordination_pipe[1])) == -1 ) {
            singularity_message(ERROR, "Other process closed write pipe unexpectedly.\n");
            ABORT(255);
        }
    }

    singularity_message(DEBUG, "Received go-ahead signal: %d\n", code);
    return(code);
}

/* Updated signal_go_ahead() which allows bi-directional wait signaling */
void singularity_signal_go_ahead(int code) {
    if ( (coordination_pipe[0] == -1) || (coordination_pipe[1] == -1)) {
        singularity_message(ERROR, "Internal error!  signal_go_ahead invoked with invalid pipe state (%d, %d).\n",
                            coordination_pipe[0], coordination_pipe[1]);
        ABORT(255);
    }

    singularity_message(DEBUG, "Sending go-ahead signal: %d\n", code);

    int retval;
    while ( (-1 == (retval = write(coordination_pipe[1], &code, 1))) && errno == EINTR) {}

    if (retval == -1) {
        if ( errno != EPIPE ) {
            singularity_message(ERROR, "Failed to send go-ahead to child process: %s (errno=%d)\n", strerror(errno), errno);
            ABORT(255);
        }
    }  // Note that we don't test for retval == 0 as we should get a EPIPE instead.

}

static int wait_child() {
    int retval, tmpstatus;
    pid_t waited_pid;

    singularity_message(DEBUG, "Parent process is waiting on child process\n");
    
    while (1) {
        /* Poll the signal handle read pipes to wait for any written signals */
        while ( -1 == (retval = poll(&fdstruct, 1, -1)) && errno == EINTR ) {}
        if ( -1 == retval ) {
            singularity_message(ERROR, "Failed to wait for file descriptors: %s\n", strerror(errno));
            ABORT(255);
        }
            
        /* Read and process the signal */
        char signum = SIGKILL;
        while (-1 == (retval = read(signal_rpipe, &signum, 1)) && errno == EINTR) {} // Flawfinder: ignore
        if (-1 == retval) {
            singularity_message(ERROR, "Failed to read from signal handler pipe: %s\n", strerror(errno));
            ABORT(255);
        }

        if ( signum != SIGCHLD ) {
            /* Forward any signal other than SIGCHLD to the child */
            singularity_message(VERBOSE2, "Forwarding signal to child: %d\n", signum);
            kill(child_pid, signum);
            continue;
        } 

        singularity_message(DEBUG, "SIGCHLD raised, waiting on the child\n");
        /* Get the pid and exit status or kill signal of the child */
        waited_pid = wait(&tmpstatus);

        if (waited_pid == child_pid) {
            singularity_message(DEBUG, "child exited, parent is exiting too\n");
            if (WIFEXITED(tmpstatus)) {
                singularity_message(DEBUG, "child exit code: %d \n", WEXITSTATUS(tmpstatus));
                return(WEXITSTATUS(tmpstatus));
            } else if (WIFSIGNALED(tmpstatus)) {
                singularity_message(DEBUG, "passing child signal to parent: %d\n", WTERMSIG(tmpstatus));
                kill(getpid(), WTERMSIG(tmpstatus));
            }
            break;
        } else {
            /* just prevented a zombie process; ignore exit code */
            singularity_message(DEBUG, "unknown child %d exited, ignoring exit code\n", waited_pid);
        }
    }

    return(-1);
}

/* */
static int clone_fn(void *data_ptr) {
    fork_state_t *state = (fork_state_t *)data_ptr;
    siglongjmp(state->env, 1);
}

/* */
static int fork_ns(unsigned int flags) {
    fork_state_t state;
    
    if ( sigsetjmp(state.env, 1) ) {
        return 0;
    }
    
    int stack_size = 1024*1024;
    void *child_stack_ptr = malloc(stack_size);
    if ( child_stack_ptr == 0 ) {
        errno = ENOMEM;
        return -1;
    }
    child_stack_ptr += stack_size;

    int retval = clone(clone_fn,
          child_stack_ptr,
          (SIGCHLD|flags),
          &state
         );
    return retval;
}

void install_signal_handle() {
    int pipes[2];
    struct sigaction action;
    sigset_t empty_mask;
    
    sigemptyset(&empty_mask);
    
    /* Fill action with handle_signal function */
    action.sa_sigaction = &handle_signal;
    action.sa_flags = SA_SIGINFO|SA_RESTART;
    action.sa_mask = empty_mask;

    singularity_message(DEBUG, "Assigning generic sigaction()s\n");
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
    if ( -1 == sigaction(SIGCHLD, &action, NULL) ) {
        singularity_message(ERROR, "Failed to install SIGCHLD signal handler: %s\n", strerror(errno));
        ABORT(255);
    }

    /* Open pipes for handle_signal() to write to */
    singularity_message(DEBUG, "Creating generic signal pipes\n");
    if ( -1 == pipe2(pipes, O_CLOEXEC) ) {
        singularity_message(ERROR, "Failed to create communication pipes: %s\n", strerror(errno));
        ABORT(255);
    }
    signal_rpipe = pipes[0];
    signal_wpipe = pipes[1];
}

pid_t singularity_fork(unsigned int flags) {
    int priv_fork = 1;
    prepare_fork();

    if ( flags == 0 || geteuid() == 0 ) {
        priv_fork = 0;
    }

    singularity_message(VERBOSE2, "Forking child process\n");
    
    if ( priv_fork == 1 ) {
        singularity_priv_escalate();
    }
    
    child_pid = fork_ns(flags);

    if ( priv_fork == 1 ) {
        singularity_priv_drop();
    }
    
    if ( child_pid == 0 ) {
        singularity_message(VERBOSE2, "Hello from child process\n");

        prepare_pipes_child();
        singularity_wait_for_go_ahead();
        
        return(child_pid);
    } else if ( child_pid > 0 ) {
        singularity_message(VERBOSE2, "Hello from parent process\n");
        prepare_pipes_parent();
        
        /* Set signal mask to block all signals while we set up sig actions */
        sigset_t blocked_mask, old_mask;
        sigfillset(&blocked_mask);
        sigprocmask(SIG_SETMASK, &blocked_mask, &old_mask);

        /* Now that we can't receive any signals, install signal handlers for all signals we want to catch */
        install_signal_handle();

        /* Set signal mask back to the original mask, unblocking the blocked signals */
        sigprocmask(SIG_SETMASK, &old_mask, NULL);

        /* Set fdstruct.fd to the read pipe created earlier */
        fdstruct.fd = signal_rpipe;
        fdstruct.events = POLLIN;
        fdstruct.revents = 0;

        /* Drop privs if we're SUID and haven't dropped permanently */
        if ( singularity_suid_enabled() && !singularity_priv_dropped_perm() ) {
            singularity_message(DEBUG, "Dropping permissions\n");
            singularity_priv_drop();
        }

        /* Allow child process to continue */
        singularity_signal_go_ahead(0);
        
        return(child_pid);
    } else {
        singularity_message(ERROR, "Failed to fork child process: %s\n", strerror(errno));
        ABORT(255);
    }    
}

void singularity_fork_run(unsigned int flags) {
    pid_t child;
    int retval;

    child = singularity_fork(flags);

    if ( child == 0 ) {
        return;
    } else if ( child > 0 ) {
        retval = wait_child();
        exit(retval);
    }
}

int singularity_fork_exec(unsigned int flags, char **argv) {
    int retval = 1;
    int i = 0;
    pid_t child;

    child = singularity_fork(0);

    if ( child == 0 ) {
        while(1) {
            if ( argv[i] == NULL ) {
                break;
            } else if ( i == 128 ) {
                singularity_message(ERROR, "singularity_fork_exec() ARGV out of bounds\n");
                ABORT(255);
            }
            singularity_message(DEBUG, "fork argv[%d] = %s\n", i, argv[i]);
            i++;
        }

        singularity_message(VERBOSE, "Running child program: %s\n", argv[0]);
        if ( execvp(argv[0], argv) < 0 ) { //Flawfinder: ignore
            singularity_message(ERROR, "Failed to exec program %s: %s\n", argv[0], strerror(errno));
            ABORT(255);
        }

    } else if ( child > 0 ) {
        retval = wait_child();
    }

    singularity_message(DEBUG, "Returning from singularity_fork_exec with: %d\n", retval);
    return(retval);
}

int singularity_fork_daemonize(unsigned int flags) {
    pid_t child;

    child = singularity_fork(flags);

    if ( child == 0 ) {
        return(0);
    } else if ( child > 0 ) {
        singularity_message(DEBUG, "Successfully spawned daemon, waiting for signal_go_ahead from child\n");

        int code = singularity_wait_for_go_ahead();
        if ( code == 0 ) {
            exit(0);
        } else {
            singularity_message(ERROR, "Daemon failed to start\n");
            ABORT(code);
        }
    }
    
    singularity_message(ERROR, "Reached unreachable code. How did you get here?\n");
    ABORT(255);

    return(0);
}
