/* 
 * Copyright (c) 2017, EDF, SA. All rights reserved.
 * 
 * This software is licensed under a 3-clause BSD license.  Please
 * consult LICENSE file distributed with the sources of this project regarding
 * your rights to use or distribute this software.
 * 
 */


#ifndef __BINARY_H_
#define __BINARY_H_

#define BINARY_ARCH_UNKNOWN 0
#define BINARY_ARCH_X86_64  1
#define BINARY_ARCH_I386    2
#define BINARY_ARCH_X32     3

int singularity_binary_arch(char* path);

#endif /* __BINARY_H_ */
