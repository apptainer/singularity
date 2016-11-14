/* Copyright (c) 2016, Michael Bauer. All rights reserved.
 *
 */

#ifndef __SINGULARITY_BOOTSTRAP_H_
#define __SINGULARITY_BOOTSTRAP_H_
    
    int singularity_bootstrap_init();

    extern int singularity_bootstrap_pre_init();
    extern int singularity_bootstrap_post_init();
    extern int singularity_bootstrap_module_init();

#endif
