/*
 * Copyright (c) 2017-2018, SyLabs, Inc. All rights reserved.
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


#ifndef __MOUNTLIST_H_
#define __MOUNTLIST_H_

// mountlist flags
#define ML_ONLY_IF_POINT_PRESENT 0x01

struct mountlist_point {
    struct mountlist_point *next;
    const char *source;
    const char *target;
    const char *filesystemtype;
    unsigned long mountflags;
    unsigned long mountlistflags;
    char *resolved_target;
};

struct mountlist {
    struct mountlist_point *first;
    struct mountlist_point *last;
};

// if source is NULL, it will be copied from target
// CONTAINER_FINALDIR will be prepended to target
// target will be freed by mountlist_cleanup, as will source if it isn't NULL
void mountlist_add(struct mountlist *mountlist,
                      const char *source, const char *target,
                      const char *filesystemtype, unsigned long mountflags,
                      unsigned long mountlistflags);
void mountlist_cleanup(struct mountlist *mountlist);

int singularity_mount_point(struct mountlist_point *point);

#endif /* __MOUNTLIST_H_ */
