/* 
 * Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
 * 
 * This software is licensed under a 3-clause BSD license.  Please
 * consult LICENSE file distributed with the sources of this project regarding
 * your rights to use or distribute this software.
 * 
 */

#ifndef __SINGULARITY_SIGNAL_H_
#define __SINGULARITY_SIGNAL_H_

void singularity_install_signal_handler();

int singularity_handle_signals(siginfo_t *siginfo);

void singularity_unblock_signals();
    

#endif
