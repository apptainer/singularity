/* 
 * Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
 *
 * Copyright (c) 2015-2017, Gregory M. Kurtzer. All rights reserved.
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

#ifndef __SINGULARITY_RUNTIME_H_
#define __SINGULARITY_RUNTIME_H_

// The Following functions actually do work:
// Unshare namespaces
extern int singularity_runtime_ns(unsigned int flags);

#define SR_NS_PID 1
#define SR_NS_IPC 2
#define SR_NS_MNT 4
#define SR_NS_NET 8
#define SR_NS_ALL 255

// Setup/initialize the overlayFS
extern int singularity_runtime_overlayfs(void);

// Setup mount points within container
extern int singularity_runtime_mounts(void);

// Setup files within the container
extern int singularity_runtime_files(void);

// Enter container root
extern int singularity_runtime_enter(void);

// Clean, santize, update environment
extern int singularity_runtime_environment(void);

// Setup for buggy autofs path
extern int singularity_runtime_autofs(void);

#endif /* __SINGULARITY_RUNTIME_H */

