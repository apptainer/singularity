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


#ifndef __SETNS_H_
#define __SETNS_H_

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
#  endif
#endif

#ifdef __NR_setns

extern int setns(int fd, int nstype);

#else /* !__NR_setns */
#  error Please determine the syscall number for setns on your architecture
#endif

#endif /* __SETNS_H_ */
