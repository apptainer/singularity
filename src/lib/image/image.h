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


extern char *singularity_image_tempdir(char *directory);
extern char *singularity_image_path(char *path);
extern char *singularity_image_name(void);

// Attach the process to a given image
extern int singularity_image_attach(void);
extern int singularity_image_attach_fd(void);
extern FILE *singularity_image_attach_fp(void);

extern int singularity_image_check(void);
extern int singularity_image_offset(void);

extern int singularity_image_bind(void);
extern char *singularity_image_bind_dev(void);

extern int singularity_image_create(unsigned int size);
extern int singularity_image_expand(unsigned int size);

extern int singularity_image_mount(unsigned int flags);
extern int singularity_image_mount_overlayfs(void);
extern char *singularity_image_mount_path(void);

#define SI_MOUNT_DEFAULTS   0
#define SI_MOUNT_RW         1
#define SI_MOUNT_DIR        2
#define SI_MOUNT_EXT4       4
#define SI_MOUNT_XFS        8
#define SI_MOUNT_SQUASHFS   16

#define LAUNCH_STRING "#!/usr/bin/env run-singularity\n"

#endif
