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

int singularity_install_signal_fd();

void singularity_handle_signals(int sig_fd);

void singularity_unblock_signals();
    

#endif
