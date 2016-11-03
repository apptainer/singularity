#ifndef __SINGULARITY_BOOTSTRAP_H_
#define __SINGULARITY_BOOTSTRAP_H_

    
    extern void singularity_bootstrap_init();

    extern int singularity_bootstrap_docker_init();
    extern int singularity_bootstrap_yum_init();
    extern int singularity_bootstrap_debootstrap_init();
    extern int singularity_bootstrap_arch_init();
    extern int singularity_bootstrap_busybox_init();

#endif
