/*
 * Copyright (c) 2017-2018, SyLabs, Inc. All rights reserved.
 * Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
 *
 * This software is licensed under a 3-clause BSD license.  Please
 * consult LICENSE.md file distributed with the sources of this project regarding
 * your rights to use or distribute this software.
 *
 */


#ifndef __SINGULARITY_CAPABILITY_H_
#define __SINGULARITY_CAPABILITY_H_

#ifdef LINUX_CAPABILITY_H

#include <linux/capability.h>

#define CAPSET_MAX  40

/* Support only 64 bits sets, since kernel 2.6.25 */
#ifdef _LINUX_CAPABILITY_VERSION_3
#  define LINUX_CAPABILITY_VERSION  _LINUX_CAPABILITY_VERSION_3
#elif defined(_LINUX_CAPABILITY_VERSION_2)
#  define LINUX_CAPABILITY_VERSION  _LINUX_CAPABILITY_VERSION_2
#else
#  error Linux 64 bits capability set not supported
#endif /* _LINUX_CAPABILITY_VERSION_3 */

int capget(cap_user_header_t, cap_user_data_t);
int capset(cap_user_header_t, const cap_user_data_t);

#endif /* LINUX_CAPABILITY_H */

#endif /* __SINGULARITY_CAPABILITY_H_ */
