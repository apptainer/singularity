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

#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <string.h>
#include <unistd.h>
#include <stdlib.h>

#include "config.h"
#include "util/util.h"
#include "util/mount.h"
#include "util/mountlist.h"
#include "util/message.h"

void mountlist_add(struct mountlist *mountlist,
                  const char *source, const char *target,
                  const char *filesystemtype, unsigned long mountflags,
                  unsigned long mountlistflags) {

    struct mountlist_point *point;
    point = (struct mountlist_point *) malloc(sizeof(struct mountlist_point));
    if (mountlist->first == NULL)
        mountlist->first = point;
    if (mountlist->last != NULL)
        mountlist->last->next = point;
    mountlist->last = point;
    point->next = NULL;
    point->source = source;
    point->target = target;
    point->filesystemtype = filesystemtype;
    point->mountflags = mountflags;
    point->mountlistflags = mountlistflags;
    point->resolved_target = NULL;
}

void mountlist_cleanup(struct mountlist *mountlist) {
    struct mountlist_point *point = mountlist->first;

    while (point != NULL) {
        if ( point->source != NULL)
            free((char *)point->source);
        if ( point->target != NULL)
            free((char *)point->target);
        if ( point->resolved_target != NULL)
            free(point->resolved_target);
        struct mountlist_point *next = point->next;
        free(point);
        point = next;
    }

    mountlist->first = NULL;
    mountlist->last = NULL;
}

int singularity_mount_point(struct mountlist_point *point) {

    int retval;
    char *target = joinpath(CONTAINER_FINALDIR, point->target);
    const char *source = point->source;
    if (source == NULL)
        source = point->target;

    retval = singularity_mount(source, target,
        point->filesystemtype, point->mountflags, NULL);

    free(target);
    return retval;
}
