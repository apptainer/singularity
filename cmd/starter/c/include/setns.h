/*
 * Copyright (c) 2017-2019, SyLabs, Inc. All rights reserved.
 * Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
 * 
 * See the COPYRIGHT.md file at the top-level directory of this distribution and at
 * https://github.com/sylabs/singularity/blob/master/COPYRIGHT.md.
 * 
 * This file is part of the Singularity Linux container project. It is subject to the license
 * terms in the LICENSE.md file found in the top-level directory of this distribution and
 * at https://github.com/sylabs/singularity/blob/master/LICENSE.md. No part
 * of Singularity, including this file, may be copied, modified, propagated, or distributed
 * except according to the terms contained in the LICENSE.md file.
 * 
*/


#ifndef __SETNS_H_
#define __SETNS_H_

#if defined(__linux__) && !defined(__NR_setns)
#  if defined(__x86_64__)
#    define __NR_setns 308
#  elif defined(__i386__)
#    define __NR_setns 346
#  elif defined(__alpha__)
#    define __NR_setns 501
#  elif defined(__arm__)
#    define __NR_setns 375
#  elif defined(__aarch64__)
#    define __NR_setns 375
#  elif defined(__ia64__)
#    define __NR_setns 1330
#  elif defined(__sparc__)
#    define __NR_setns 337
#  elif defined(__powerpc__)
#    define __NR_setns 350
#  elif defined(__s390__) || defined(__s390x__)
#    define __NR_setns 339
#  elif defined(__mips__)
#    if _MIPS_SIM == _ABIO32
#      define __NR_setns 4344
#    elif _MIPS_SIM == _ABI64
#      define __NR_setns 5303
#    elif _MIPS_SIM == _ABIN32
#      define __NR_setns 6308
#    endif
#  elif defined(__m68k__)
#    define __NR_setns 344
#  elif defined(__hppa__)
#    define __NR_setns 328
#  elif defined(__sh__) && (!defined(__SH5__) || __SH5__ == 32)
#    define __NR_setns 364
#  elif defined(__sh__) && defined(__SH5__) && __SH5__ == 64
#    define __NR_setns 375
#  elif defined(__bfin__)
#    define __NR_setns 379
#  else
#    error Please determine the syscall number of setns on your architecture
#  endif
#endif

#endif /* __SETNS_H_ */
