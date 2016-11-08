#ifndef __SINGULARITY_BOOTSTRAP_MODULE_H_
#define __SINGULARITY_BOOTSTRAP_MODULE_H_

    int singularity_bootstrap_module_init();
    extern int singularity_bootstrap_docker_init();
    extern int singularity_bootstrap_yum_init();
    extern int singularity_bootstrap_arch_init();
    extern int singularity_bootstrap_debootstrap_init();
    extern int singularity_bootstrap_busybox_init();

#endif
