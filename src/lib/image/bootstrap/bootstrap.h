/* Copyright (c) 2016, Michael Bauer. All rights reserved.
 *
 */

#ifndef __SINGULARITY_BOOTSTRAP_H_
#define __SINGULARITY_BOOTSTRAP_H_
    
    int singularity_bootstrap(int argc, char ** argv);
    void singularity_bootstrap_script_run(char *section_name);

    int bootstrap_module_init();
    int bootstrap_rootfs_install();
    int bootstrap_copy_defaults();
    void bootstrap_copy_runscript();

    extern int singularity_bootstrap_docker();
    extern int singularity_bootstrap_yum();
    extern int singularity_bootstrap_debootstrap();
    extern int singularity_bootstrap_busybox();
    extern int singularity_bootstrap_arch();

#endif
