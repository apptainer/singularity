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

#ifndef __SINGULARITY_SIGNAL_HANDLERS_H_
#define __SINGULARITY_SIGNAL_HANDLERS_H_

#include <sys/types.h>

// Setup all the necessary signal handlers.
void setup_signal_handler(pid_t pid);
// Block until the given PID exits (but _don't_ call wait() on it),
// but also process signal handler activity in the meantime.
// On return, waitpid should be invoked on pid.
//
// If an "interesting" signal is caught while we're in this function,
// then we'll forward it to `pid` via kill(), then return.
void blockpid_or_signal();

// Setup the communication pipes for monitoring the status of the parent.
void signal_pre_fork();
void signal_post_child();
void signal_post_parent();

#endif // __SINGULARITY_SIGNAL_HANDLERS_H_
