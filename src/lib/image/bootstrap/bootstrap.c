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

static char *module_name;
static char *rootfs_path;

int singularity_bootstrap_init(int argc, char ** argv) {

  char *bootdef_path;
  char *driver_v1_path = LIBEXECDIR "/singularity/bootstrap/driver-v1.sh";

  if ( argv[1] == NULL ) {
    fprintf(stderr, "USAGE: UPDATE USAGE HERE!! SINGULARITY_IMAGE=[image] %s [command...]\n", argv[0]);
    return(1);
  }

  //Abort if we can't open the bootstrap definition file
  if( singularity_bootdef_open(argv[1]) != 0 ) {
    ABORT(255);
  }

  //image-mount has finished, we are now inside a fork of image-mount running image-bootstrap binary instead of bootstrap.sh

  //mktemp -d /tmp/singularity-bootstrap.XXXXXXX ?? Unsure where this was used but was in the bootstrap shell scripts

  //Execute old driver-v1.sh if bootdef_ver = 1, else execute new bootstrap code
  if( ( bootstrap_ver = singularity_bootdef_get_version() ) == 1 ) {
    singularity_message(INFO, "Running bootstrap driver v1\n");
    singularity_bootdef_close();

    //argv[0] = driver_v1_path; //pointer to some string containing path to driver-v1.sh script
    //singularity_fork_exec(argv); //Use singularity_fork_exec to directly call the v1 driver
    return(0);

    //Maybe directly use driver-v2.sh since it is outdated and we don't need to rewrite it for future use?
  } else {
    singularity_message(DEBUG, "Running bootstrap driver v2\n");

    singularity_bootstrap_script_run("pre"); //Replaces prebootstrap file since it does nothing else

    singularity_bootstrap_module_init(); //lib/bootstrap/bootstrap.c

    //singularity_postbootstrap_init(); //lib/bootstrap/postbootstrap/postbootstrap.c

    /* Next section here replaces postbootstrap module */
    rootfs_path = singularity_rootfs_dir();
    singularity_rootfs_check();
    
    if ( postbootstrap_rootfs_install() != 0 ) {
      singularity_message(ERROR, "Failed to create container rootfs. Aborting...\n");
      ABORT(255);
    }
    
    postbootstrap_copy_runscript();
    
    if ( postbootstrap_copy_defaults() != 0 ) {
      singularity_message(ERROR, "Failed to copy necessary default files to container rootfs. Aborting...\n");
      ABORT(255);
    }
    
    singularity_bootstrap_script_run("setup");
      
    singularity_rootfs_chroot();
    singularity_bootstrap_script_run("post");
    
    singularity_bootdef_close();   
  }
  return(0);
}

void singularity_bootstrap_script_run(char *section_name) {
  char ** fork_args;
  char *script;
  char *args;
  int retval;

  fork_args = malloc(sizeof(char *) * 3);

  singularity_message(VERBOSE, "Searching for %%%s bootstrap script\n", section_name);
  if ( ( args = singularity_bootdef_section_get(script, section_name) ) == NULL ) {
    singularity_message(VERBOSE, "No %%%s bootstrap script found, skipping\n", section_name);
    return;
  } else {
    fork_args[0] = strdup("/bin/sh");
    //fork_args[1] = strdup("-e -x"); //Will handle in section_get
    fork_args[1] = args;
    fork_args[2] = script;
    singularity_message(INFO, "Running %%%s bootstrap script\n", section_name);

    if ( ( retval = singularity_fork_exec(fork_args) ) != 0 ) {
      singularity_message(WARNING, "Something may have gone wrong. %%%s script exited with status: %i\n", section_name, retval);
    }
    free(fork_args[0]);
    free(fork_args[1]);
    free(fork_args[2]);
    free(fork_args);
  }
}

int singularity_bootstrap_module_init() {
  singularity_bootdef_rewind();

  if ( ( module_name = singularity_bootdef_get_value("Bootstrap") ) == NULL ) {
    singularity_message(ERROR, "Bootstrap definition file does not contain a Bootstrap: line");
    ABORT(255);

  } else {
    singularity_message(INFO, "Running bootstrap module %s\n", module_name);

    if ( strcmp(module_name, "docker") ) { //Docker
      return( singularity_bootstrap_docker_init() );

    } else if ( strcmp(module_name, "yum") ) { //Yum
      return( singularity_bootstrap_yum_init() );

    } else if ( strcmp(module_name, "debootstrap") ) { //Debootstrap
      return( singularity_bootstrap_debootstrap_init() );

    } else if ( strcmp(module_name, "arch") ) { //Arch
      return( singularity_bootstrap_arch_init() );

    } else if ( strcmp(module_name, "busybox") ) { //Busybox
      return( singularity_bootstrap_busybox_init() );

    } else {
      singularity_message(ERROR, "Could not parse bootstrap module of type: %s", module_name);
      ABORT(255);
    }

  }
}

int postbootstrap_rootfs_install() {
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
  retval += mount("/proc/", joinpath(rootfs_path, "/proc"), "proc", NULL, NULL);
  retval += mount("/sys/", joinpath(rootfs_path, "/sys"), "sysfs", NULL, NULL);
  retval += mount("/dev/", joinpath(rootfs_path, "/dev"), /* Type of /dev/ */, MS_REMOUNT, NULL);

  return(retval);

}

int postbootstrap_copy_defaults() {
  int retval = 0;

  if ( is_file(joinpath(rootfs_path, "/environment")) ) {
    singularity_message(INFO, "Skipping environment file, file already exists.\n");
  } else {
    retval += copy_file( LIBEXECDIR "/singularity/defaults/environment", joinpath(rootfs_path, "/environment") );
  }
  retval += copy_file( LIBEXECDIR "/singularity/defaults/exec", joinpath(rootfs_path, "/.exec") );
  retval += copy_file( LIBEXECDIR "/singularity/defaults/shell", joinpath(rootfs_path, "/.shell") );
  retval += copy_file( LIBEXECDIR "/singularity/defaults/run", joinpath(rootfs_path, "/.run") );
  retval += copy_file( "/etc/hosts", joinpath(rootfs_path, "/etc/hosts") );
  retval += copy_file( "/etc/resolv.conf", joinpath(rootfs_path, "/etc/resolv.conf") );


  return(retval);
}

void postbootstrap_copy_runscript() {
  char *script;

  if ( singularity_bootdef_section_get(script, "runscript") == NULL ) {
    singularity_message(VERBOSE, "Definition file does not contain runscript, skipping.\n");
    return;
  }

  if ( fileput(joinpath(rootfs_path, "/singularity"), script) < 0 ) {
    singularity_message(WARNING, "Couldn't write to rootfs/singularity, skipping runscript.\n");
  }
}
