/* 
 * Copyright (c) 2015-2016, Gregory M. Kurtzer. All rights reserved.
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

#ifndef __SINGULARITY_RUNTIME_H_
#define __SINGULARITY_RUNTIME_H_

// Set and return the runtime container directory location to use. If
// 'directory' is NULL, then it will return the currently set directory.
char *singularity_runtime_dir(char *directory);

// Set the runtime flags (below). Flags can be combined using a bitwise OR.
int singularity_runtime_flags(unsigned int flags);

// Each of the below will cascade down the modules activatng each of the
// primary interface drivers:
//
//  check:      Make sure the environment is such that it is ready to run
//  prepare:    Any presetup functions that need to happen before acivation
//  activate:   Run the module
extern int singularity_runtime_check(void);
extern int singularity_runtime_prepare(void);
extern int singularity_runtime_activate(void);

// Runtime flags
#define SR_FLAGS        0   // Do not make any changes and return flags
#define SR_NOSUID       1   // We are not running SUID
#define SR_NOFORK       2   // Do not allow forking

#endif /* __SINGULARITY_RUNTIME_H */

