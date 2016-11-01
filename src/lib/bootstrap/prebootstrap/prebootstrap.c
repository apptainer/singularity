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

void singularity_prebootstrap_init() {
  
  singularity_prebootstrap_set_rootfs();
  singularity_prebootstrap_install_rootfs();
  singularity_prebootstrap_run_script();
  
}

void singularity_prebootstrap_install_rootfs() {
  s_mkpath(rootfs_path, 0755);
  s_mkpath(strjoin(rootfs, "/dev"), 0755);


  //Do this in C
  cp -a /dev/null         "$SINGULARITY_ROOTFS/dev/null"      2>/dev/null || > "$SINGULARITY_ROOTFS/dev/null";
  cp -a /dev/zero         "$SINGULARITY_ROOTFS/dev/zero"      2>/dev/null || > "$SINGULARITY_ROOTFS/dev/zero";
  cp -a /dev/random       "$SINGULARITY_ROOTFS/dev/random"    2>/dev/null || > "$SINGULARITY_ROOTFS/dev/random";
  cp -a /dev/urandom      "$SINGULARITY_ROOTFS/dev/urandom"   2>/dev/null || > "$SINGULARITY_ROOTFS/dev/urandom";

  
}

void singularity_prebootstrap_set_rootfs() {
  if( rootfs_path == NULL ) {
    rootfs_path = envar_path(rootfs_envar);
  }
}

void singularity_prebootstrap_run_script() {
  char *pre_script;
  char *section_name = "pre";

  singularity_bootdef_section_get(pre_script, section_name);

  
  
