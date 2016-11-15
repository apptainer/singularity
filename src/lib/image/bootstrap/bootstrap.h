/* Copyright (c) 2016, Michael Bauer. All rights reserved.
 *
 */

#ifndef __SINGULARITY_BOOTSTRAP_H_
#define __SINGULARITY_BOOTSTRAP_H_
    
    int singularity_bootstrap_init(int argc, char ** argv);
    void singularity_bootstrap_script_run(char *section_name);

    int bootstrap_module_init();
    int bootstrap_rootfs_install();
    int bootstrap_copy_defaults();
    void bootstrap_copy_runscript();

#endif
