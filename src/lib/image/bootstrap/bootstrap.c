/* Copyright (c) 2016, Michael Bauer. All rights reserved.
 *
 */

#define _GNU_SOURCE
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

int singularity_bootstrap(int argc, char ** argv) {
  char *containerimage;
  char *driver_v1_path = LIBEXECDIR "/singularity/bootstrap/driver-v1.sh";
  singularity_message(VERBOSE, "Preparing to bootstrap image with definition file: %s\n", argv[1]);
  
  /* Sanity check on input */
  if ( argv[1] == NULL ) {
    singularity_message(ERROR, "singularity_bootstrap expects argv[1] to contain file, found NULL\n");
    return(1);
  }

  /* Error out if $SINGULARITY_IMAGE is not defined */
  singularity_message(DEBUG, "Obtaining container name from environment variable\n");
  if ( ( containerimage = envar_path("SINGULARITY_IMAGE") ) == NULL ) {
    singularity_message(ERROR, "SINGULARITY_IMAGE not defined!\n");
    ABORT(255);
  }

  /* Sanity check to ensure we can properly open the bootstrap definition file */
  singularity_message(DEBUG, "Opening singularity bootdef file: %s\n", argv[1]);
  setenv("SINGULARITY_BUILDDEF", argv[1], 1);
  if( singularity_bootdef_open(argv[1]) != 0 ) {
    singularity_message(ERROR, "Could not open bootdef file\n");
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
  setenv("SINGULARITY_ROOTFS", singularity_rootfs_dir(), 1);

  /* Determine if Singularity file is v1 or v2. v1 files will directly use the old driver-v1.sh script */
  if( singularity_bootdef_get_version() == 1 ) {
    singularity_message(VERBOSE, "Running bootstrap driver v1\n");
    singularity_bootdef_close();
    
    /* Directly call on old driver-v1.sh */
    argv[0] = driver_v1_path; //pointer to some string containing path to driver-v1.sh script
    singularity_fork_exec(argv); //Use singularity_fork_exec to directly call the v1 driver
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
    
    bootstrap_copy_runscript();

    singularity_file();
    if ( bootstrap_copy_defaults() != 0 ) {
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
  char *args;
  int retval;

  fork_args = malloc(sizeof(char *) * 4);
  script = malloc(sizeof(char *));
  
  singularity_message(VERBOSE, "Searching for %%%s bootstrap script\n", section_name);
  if ( ( args = singularity_bootdef_section_get(script, section_name) ) == NULL ) {
    singularity_message(VERBOSE, "No %%%s bootstrap script found, skipping\n", section_name);
    return;
  } else {
    
    fork_args[0] = strdup("/bin/sh");
    fork_args[1] = strdup("-c");
    fork_args[2] = *script;
    fork_args[3] = NULL;
    singularity_message(VERBOSE, "Running %%%s bootstrap script\n%s\n%s\n", section_name, fork_args[1], fork_args[2]);

    if ( ( retval = singularity_fork_exec(fork_args) ) != 0 ) {
      singularity_message(WARNING, "Something may have gone wrong. %%%s script exited with status: %i\n", section_name, retval);
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

  return(retval);

}

/*
 * Copies .exec & .shell & .run into container rootfs. Copies environment file
 * if no environment file is already present.
 *
 * @returns 0 on success, <0 on failure
 */
int bootstrap_copy_defaults() {
  singularity_message(VERBOSE, "Copying default files into container rootfs.\n");
  int retval = 0;
  char *helper_shell;

  if ( is_file(joinpath(rootfs_path, "/bin/bash")) == 0 ) {
    helper_shell = strdup("#!/bin/bash");
  } else {
    helper_shell = strdup("#!/bin/sh");
  }

  if ( is_file(joinpath(rootfs_path, "/environment")) == 0 ) {
    singularity_message(VERBOSE, "Skipping environment file, file already exists.\n");
  } else {
    singularity_message(DEBUG, "Copying /environment into container rootfs\n");
    retval += copy_file( LIBEXECDIR "/singularity/defaults/environment", joinpath(rootfs_path, "/environment") );
    retval += chmod( joinpath(rootfs_path, "/environment"), 0644 );
  }

  singularity_message(DEBUG, "Copying /.exec, /.shell, && /.run into container rootfs\n");
  retval += fileput(joinpath(rootfs_path, "/.exec"), strjoin(helper_shell, filecat(LIBEXECDIR "/singularity/defaults/exec")));
  retval += chmod( joinpath(rootfs_path, "/.exec"), 0755 );
  
  retval += fileput(joinpath(rootfs_path, "/.shell"), strjoin(helper_shell, filecat(LIBEXECDIR "/singularity/defaults/shell")));
  retval += chmod( joinpath(rootfs_path, "/.shell"), 0755 );
  
  retval += fileput(joinpath(rootfs_path, "/.run"), strjoin(helper_shell, filecat(LIBEXECDIR "/singularity/defaults/run")));
  retval += chmod( joinpath(rootfs_path, "/.run"), 0755 );

  free(helper_shell);
  return(retval);
}

/*
 * Copies %runscript as defined in bootstrap spec file into container rootfs.
 * Verbose output will inform user if no runscript was found.
 *
 * @returns nothing
 */
void bootstrap_copy_runscript() {
  char **script = malloc(sizeof(char *));
  singularity_message(DEBUG, "Searching for runscript in definition file.\n");

  if ( singularity_bootdef_section_get(script, "runscript") == NULL ) {
    singularity_message(VERBOSE, "Definition file does not contain runscript, skipping.\n");
    free(script);
    return;
  }

  if ( fileput(joinpath(rootfs_path, "/singularity"), *script) < 0 ) {
    singularity_message(WARNING, "Couldn't write to rootfs/singularity, skipping runscript.\n");
  }
  
  free(*script);
  free(script);
}
