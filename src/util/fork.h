/* 
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

#ifndef __SINGULARITY_FORK_H_
#define __SINGULARITY_FORK_H_


    // SINGULARITY_FORK()
    // Wrap the fork() system call and create the necessary communication
    // pipes and signal handlers so that signals are correctly passed around
    // between children and parents.
    pid_t singularity_fork(void);


    // SINGLARITY_FORK_RUN()
    // Fork() and automatically have the parent wait on the child while
    // allowing the child to continue on through the code path. The parent
    // will automatically wait in the background until the child exits, and
    // then the parent will also exit with the same exit code as the parent.
    // Similar to singularity_fork() above, this will maintain the proper
    // communication channels for signal handling.
    void singularity_fork_run(void);


    // SINGULARITY_FORK_EXEC
    // Fork and exec a child system command, wait for it to return, and then
    // return with the appropriate exit value.
    int singularity_fork_exec(char **argv);


#endif /* __SINGULARITY_FORK_H_ */
