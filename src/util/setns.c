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
#include <errno.h>
#include <sys/syscall.h>

#include "util/message.h"

#if defined (SINGULARITY_NO_SETNS) && defined (SINGULARITY_SETNS_SYSCALL)

#ifndef __NR_setns
#  if defined (__x86_64__)
#    define __NR_setns 308
#  elif defined (__i386__)
#    define __NR_setns 346
#  elif defined (__alpha__)
#    define __NR_setns 501
#  elif defined (__arm__)
#    define __NR_setns 375
#  elif defined (__aarch64__)
#    define __NR_setns 375
#  elif defined (__ia64__)
#    define __NR_setns 1330
#  elif defined (__sparc__)
#    define __NR_setns 337
#  elif defined (__powerpc__)
#    define __NR_setns 350
#  elif defined (__s390__)
#    define __NR_setns 339
#  else
#    error Please determine the syscall number for setns on your architecture
#  endif
#endif

int setns(int fd, int nstype) {
    singularity_message(DEBUG, "Using syscall() wrapped __NR_setns\n");
    return syscall(__NR_setns, fd, nstype);
}

#elif defined (SINGULARITY_NO_SETNS) && !defined (SINGULARITY_SETNS_SYSCALL)

int setns(int fd, int nstype) {
    singularity_message(VERBOSE, "setns() not supported at compile time by kernel at time of building\n");
    errno = ENOSYS;
    return -1;
}

#endif /* SINGULARITY_NO_SETNS && SINGULARITY_SETNS_SYSCALL */
