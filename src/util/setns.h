/* 
 * Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
 *
 * Copyright (c) 2015-2017, Gregory M. Kurtzer. All rights reserved.
 * 
 * Copyright (c) 2016-2017, The Regents of the University of California,
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
