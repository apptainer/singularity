/* Copyright (c) 2016, Michael Bauer. All rights reserved.
 *
 */

#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/mount.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <sys/param.h>
#include <errno.h>
#include <signal.h>
#include <sched.h>
#include <string.h>
#include <fcntl.h>
#include <grp.h>
#include <libgen.h>
#include <linux/limits.h>

#include "config.h"
#include "lib/singularity.h"
#include "util/file.h"
#include "util/util.h"

static char *rootfs_envar = "SINGULARITY_ROOTFS";
static char *rootfs_path = NULL;

void singularity_postbootstrap_init() {
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
