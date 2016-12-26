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


#ifndef __SINGULARITY_IMAGE_H_
#define __SINGULARITY_IMAGE_H_

// Attach the process to a given image
extern int singularity_image_attach(char *path);
extern int singularity_image_attach_fd();
extern FILE *singularity_image_attach_fp();

extern int singualrity_image_check(FILE *image_fp);
extern int singualrity_image_offset(FILE *image_fp);

extern int singularity_image_bind(FILE *image_fp);
extern char *singularity_image_bind_dev();

extern int singularity_image_create(char *image, unsigned int size);
extern int singularity_image_expand(FILE *image_fp, unsigned int size);

extern int singularity_image_mount(char *mountpoint, unsigned int flags);

#define SI_MOUNT_DEFAULTS   0
#define SI_MOUNT_RW         1
#define SI_MOUNT_DIR        2
#define SI_MOUNT_EXT4       4
#define SI_MOUNT_XFS        8
#define SI_MOUNT_SQUASHFS   16

#define LAUNCH_STRING "#!/usr/bin/env run-singularity\n"

#endif
