/* 
 * Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
 * 
 * This software is licensed under a 3-clause BSD license.  Please
 * consult LICENSE file distributed with the sources of this project regarding
 * your rights to use or distribute this software.
 * 
 */


#ifndef __CAPABILITY_H_
#define __CAPABILITY_H_

void singularity_capability_init(void);
void singularity_capability_init_minimal(void);
void singularity_capability_init_default(void);

// Drop all capabilities
void singularity_capability_drop(void);

#endif /* __CAPABILITY_H_ */
