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

#ifndef __SINGULARITY_RUNTIME_ROOTFS_CHROOT_H_
#define __SINGULARITY_RUNTIME_ROOTFS_CHROOT_H_

extern int singularity_runtime_rootfs_chroot_precheck(void);
extern int singularity_runtime_rootfs_chroot_setup(void);
extern int singularity_runtime_rootfs_chroot_activate(void);
extern int singularity_runtime_rootfs_chroot_contain(void);

#endif /* __SINGULARITY_RUNTIME_ROOTFS_CHROOT_H */

