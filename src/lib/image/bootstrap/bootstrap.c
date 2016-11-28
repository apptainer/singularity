/* 
 * Copyright (c) 2016, Michael W. Bauer. All rights reserved.
 * 
 * “Singularity” Copyright (c) 2016, The Regents of the University of California,
 * through Lawrence Berkeley National Laboratory (subject to receipt of any
 * required approvals from the U.S. Dept. of Energy).  All rights reserved.
 * 
 * This software is licensed under a customized 3-clause BSD license.  Please
 * consult LICENSE file distributed with the sources of this project regarding
 * your rights to use or distribute this software.
 * 
 * NOTICE.  This Software was developed under funding from the U.S. Department of
 * Energy and the U.S. Government consequently retains certain rights. As such,
 * the U.S. Government has been granted for itself and others acting on its
 * behalf a paid-up, nonexclusive, irrevocable, worldwide license in the Software
 * to reproduce, distribute copies to the public, prepare derivative works, and
 * perform publicly and display publicly, and to permit other to do so. 
 * 
 */

#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <string.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/mount.h>
#include <sys/wait.h>
#include <unistd.h>
#include <stdlib.h>
#include <sched.h>

#include "util/file.h"
#include "util/util.h"
#include "lib/message.h"
#include "lib/singularity.h"
#include "lib/image/bootstrap/bootdef_parser.h"
#include "lib/image/bootstrap/bootstrap.h"


static char *module_name;
static char *rootfs_path;


int singularity_bootstrap(char *containerimage, char *bootdef_path) {
    char *driver_v1_path = LIBEXECDIR "/singularity/bootstrap/driver-v1.sh";
    singularity_message(VERBOSE, "Preparing to bootstrap image with definition file: %s\n", bootdef_path);

    /* Sanity check to ensure we can properly open the bootstrap definition file */
    singularity_message(DEBUG, "Opening singularity bootdef file: %s\n", bootdef_path);
    if( singularity_bootdef_open(bootdef_path) != 0 ) {
        singularity_message(ERROR, "Could not open bootstrap definition file\n");
        ABORT(255);
    }

    /* Initialize namespaces and session directory on the host */
    singularity_message(DEBUG, "Initializing container directory\n");
    singularity_sessiondir_init(containerimage);
    singularity_ns_user_unshare();
    singularity_ns_mnt_unshare();

    /* Initialize container rootfs directory and corresponding variables */
    singularity_message(DEBUG, "Mounting container rootfs\n");
    singularity_rootfs_init(containerimage);
    singularity_rootfs_mount();
    rootfs_path = singularity_rootfs_dir();
    
    /* Set environment variables required for any shell scripts we will call on */
    setenv("SINGULARITY_ROOTFS", singularity_rootfs_dir(), 1);
    setenv("SINGULARITY_IMAGE", containerimage, 1);
    setenv("SINGULARITY_BUILDDEF", bootdef_path, 1);

    /* Determine if Singularity file is v1 or v2. v1 files will directly use the old driver-v1.sh script */
    if( singularity_bootdef_get_version() == 1 ) {
        singularity_message(VERBOSE, "Running bootstrap driver v1\n");
        singularity_bootdef_close();
    
        /* Directly call on old driver-v1.sh */
        singularity_fork_exec(&driver_v1_path); //Use singularity_fork_exec to directly call the v1 driver
        return(0);

    } else {
        singularity_message(VERBOSE, "Running bootstrap driver v2\n");
    
        /* Run %pre script to replace prebootstrap module */    
        singularity_bootstrap_script_run("pre");

        /* Run appropriate module to create the base OS in the container */
        if ( bootstrap_module_init() != 0 ) {
            singularity_message(ERROR, "Something went wrong during build module. \n");
        }

        /* Run through postbootstrap module logic */
    
        /* Ensure that rootfs has required folders, permissions and files */
        singularity_rootfs_check();
    
        if ( bootstrap_rootfs_install() != 0 ) {
            singularity_message(ERROR, "Failed to create container rootfs. Aborting...\n");
            ABORT(255);
        }
    
        bootstrap_copy_script("runscript", "/singularity");
        if ( bootstrap_copy_script("environment", "/environment") != 0 ) {
            singularity_message(VERBOSE, "Copying default environment file instead of user specified environment\n");
            copy_file(LIBEXECDIR "/singularity/defaults/environment", joinpath(rootfs_path, "/environment"));
        }
        chmod(joinpath(rootfs_path, "/environment"), 0644);
            

        /* Copy/mount necessary files directly into container rootfs */
        if ( singularity_file_bootstrap() < 0 ) {
            singularity_message(ERROR, "Failed to copy necessary default files to container rootfs. Aborting...\n");
            ABORT(255);
        }

        /* Mount necessary folders into container */
        if ( singularity_mount() < 0 ) {
            singularity_message(ERROR, "Failed to mount necessary files into container rootfs. Aborting...\n");
            ABORT(255);
        }

        /* Run %setup script from host */
        singularity_bootstrap_script_run("setup");

        /* Run %post script from inside container */
        singularity_rootfs_chroot();
        singularity_bootstrap_script_run("post");
    
        singularity_bootdef_close();   
    }
    return(0);
}

/*
 * Runs the specified script within the bootstrap spec file. Forks a child process and waits
 * until that process terminates to continue in the main process thread.
 *
 * @param char *section_name pointer to string containing section name of script to run
 * @returns nothing
 */
void singularity_bootstrap_script_run(char *section_name) {
    char **fork_args;
    char **script;

    fork_args = malloc(sizeof(char *) * 4);
    script = malloc(sizeof(char *));
  
    singularity_message(VERBOSE, "Searching for %%%s bootstrap script\n", section_name);
    if ( singularity_bootdef_section_get(script, section_name) == -1 ) {
        singularity_message(VERBOSE, "No %%%s bootstrap script found, skipping\n", section_name);
        return;
    } else {
    
        fork_args[0] = strdup("/bin/sh");
        fork_args[1] = strdup("-c");
        fork_args[2] = *script;
        fork_args[3] = NULL;
        singularity_message(VERBOSE, "Running %%%s bootstrap script\n%s%s\n%s\n", section_name, fork_args[0], fork_args[1], fork_args[2]);

        if ( singularity_fork_exec(fork_args) != 0 ) {
            singularity_message(WARNING, "Something may have gone wrong. %%%s script exited with non-zero status.\n", section_name);
        }
        free(fork_args[0]);
        free(fork_args[1]);
        free(fork_args[2]);
        free(fork_args[3]);
        free(fork_args);
    }
}

/*
 * Determines which module the bootstrap spec file belongs to and runs the appropriate workflow.
 *
 * @returns 0 on success, -1 on failure
 */
int bootstrap_module_init() {
    singularity_bootdef_rewind();

    if ( ( module_name = singularity_bootdef_get_value("BootStrap") ) == NULL ) {
        singularity_message(ERROR, "Bootstrap definition file does not contain required Bootstrap: option\n");
        return(-1);

    } else {
        singularity_message(VERBOSE, "Running bootstrap module %s\n", module_name);

        if ( strcmp(module_name, "docker") == 0 ) { //Docker
            return( singularity_bootstrap_docker() );

        } else if ( strcmp(module_name, "yum") == 0) { //Yum
            return( singularity_bootstrap_yum() );

        } else if ( strcmp(module_name, "debootstrap") == 0 ) { //Debootstrap
            return( singularity_bootstrap_debootstrap() );

        } else if ( strcmp(module_name, "arch") == 0 ) { //Arch
            return( singularity_bootstrap_arch() );

        } else if ( strcmp(module_name, "busybox") == 0 ) { //Busybox
            return( singularity_bootstrap_busybox() );

        } else {
            singularity_message(ERROR, "Could not parse bootstrap module of type: %s\n", module_name);
            return(-1);
        }

    }
}

/*
 * Ensures that the paths are properly installed with correct permissions.
 *
 * @returns 0 on success, <0 on failure
 */
int bootstrap_rootfs_install() {
    int retval = 0;
    retval += s_mkpath(rootfs_path, 0755);
    retval += s_mkpath(joinpath(rootfs_path, "/bin"), 0755);
    retval += s_mkpath(joinpath(rootfs_path, "/dev"), 0755);
    retval += s_mkpath(joinpath(rootfs_path, "/home"), 0755);
    retval += s_mkpath(joinpath(rootfs_path, "/etc"), 0755);
    retval += s_mkpath(joinpath(rootfs_path, "/root"), 0750);
    retval += s_mkpath(joinpath(rootfs_path, "/proc"), 0755);
    retval += s_mkpath(joinpath(rootfs_path, "/sys"), 0755);
    retval += s_mkpath(joinpath(rootfs_path, "/tmp"), 1777);
    retval += s_mkpath(joinpath(rootfs_path, "/var/tmp"), 1777);
    retval += copy_file("/etc/hosts", joinpath(rootfs_path, "/etc/hosts"));
    retval += copy_file("/etc/resolv.conf", joinpath(rootfs_path, "/etc/resolv.conf"));
    unlink(joinpath(rootfs_path, "/etc/mtab"));
    retval += fileput(joinpath(rootfs_path, "/etc/mtab"), "singularity / rootfs rw 0 0");

    return(retval);

}

/*
 * Copies script given by section_name into file in container rootfs given
 * by dest_path.
 *
 * @param char *section_name pointer to string containing name of section to copy
 * @param char *dest_path pointer to string containing path to copy script into
 * @returns nothing
 */
int bootstrap_copy_script(char *section_name, char *dest_path) {
    char **script = malloc(sizeof(char *));
    char *full_dest_path = joinpath(rootfs_path, dest_path);
    singularity_message(VERBOSE, "Attempting to copy %%%s script into %s in container.\n", section_name, dest_path);
    
    if ( singularity_bootdef_section_get(script, section_name) == -1 ) {
        singularity_message(VERBOSE, "Definition file does not contain %s, skipping.\n", section_name);
        free(full_dest_path);
        free(script);
        return(-1);
    }
    
    if ( fileput(full_dest_path, *script) < 0 ) {
        singularity_message(WARNING, "Couldn't write to %s, skipping %s.\n", full_dest_path, section_name);
        free(full_dest_path);
        free(*script);
        free(script);
        return(-1);
    }

    chmod(full_dest_path, 0755);

    free(full_dest_path);
    free(*script);
    free(script);
    return(0);
}
