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
  

  
  postbootstrap_script_run("setup");

  singularity_rootfs_chroot();
  postbootstrap_script_run("post");
}

void postbootstrap_rootfs_install() {
  s_mkpath(rootfs_path, 0755);
  s_mkpath(joinpath(rootfs_path, "/bin"), 0755);
  s_mkpath(joinpath(rootfs_path, "/dev"), 0755);
  s_mkpath(joinpath(rootfs_path, "/home"), 0755);
  s_mkpath(joinpath(rootfs_path, "/etc"), 0755);
  s_mkpath(joinpath(rootfs_path, "/root"), 0750);
  s_mkpath(joinpath(rootfs_path, "/proc"), 0755);
  s_mkpath(joinpath(rootfs_path, "/sys"), 0755);
  s_mkpath(joinpath(rootfs_path, "/tmp"), 1777);
  s_mkpath(joinpath(rootfs_path, "/var/tmp"), 1777);
  

  //Do this in C (Or maybe move this to yum.c since it was changed in upstream/master??)
  //cp -a /dev/null         "$SINGULARITY_ROOTFS/dev/null"      2>/dev/null || > "$SINGULARITY_ROOTFS/dev/null";
  //cp -a /dev/zero         "$SINGULARITY_ROOTFS/dev/zero"      2>/dev/null || > "$SINGULARITY_ROOTFS/dev/zero";
  //cp -a /dev/random       "$SINGULARITY_ROOTFS/dev/random"    2>/dev/null || > "$SINGULARITY_ROOTFS/dev/random";
  //cp -a /dev/urandom      "$SINGULARITY_ROOTFS/dev/urandom"   2>/dev/null || > "$SINGULARITY_ROOTFS/dev/urandom";

  
}

void postbootstrap_script_run(char *section_name) {
  char ** script;
  //char *section_name = "post";
  char *args;
  singularity_message(VERBOSE, "Searching for %%%s bootstrap script\n", section_name);
  if ( ( args = singularity_bootdef_section_get(script, section_name) ) == NULL ) {
    singularity_message(VERBOSE, "No %%%s bootstrap script found, skipping\n", section_name);
    return;
  } else {
    singularity_message(INFO, "Running %%%s bootstrap script on host\n", section_name);
    singularity_fork_exec() //use this to execute the script with the arguments and commands

  }
}
