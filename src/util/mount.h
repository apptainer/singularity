/*
 * Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
 * 
 * See the COPYRIGHT.md file at the top-level directory of this distribution and at
 * https://github.com/singularityware/singularity/blob/master/COPYRIGHT.md.
 * 
 * This file is part of the Singularity Linux container project. It is subject to the license
 * terms in the LICENSE.md file found in the top-level directory of this distribution and
 * at https://github.com/singularityware/singularity/blob/master/LICENSE.md. No part
 * of Singularity, including this file, may be copied, modified, propagated, or distributed
 * except according to the terms contained in the LICENSE.md file.
 * 
*/


#ifndef __MOUNT_H_
#define __MOUNT_H_

int singularity_mount(const char *source, const char *target,
                      const char *filesystemtype, unsigned long mountflags,
                      const void *data);
int check_mounted(char *mountpoint);

#endif /* __MOUNT_H_ */
