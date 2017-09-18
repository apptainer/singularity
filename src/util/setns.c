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


#define _GNU_SOURCE

#include <unistd.h>
#include <sys/syscall.h>

#if defined (NO_SETNS) && defined (SINGULARITY_SETNS_SYSCALL)

#include "util/setns.h"

int setns(int fd, int nstype) {
    return syscall(__NR_setns, fd, nstype);
}

#endif /* NO_SETNS && SINGULARITY_SETNS_SYCALL */
