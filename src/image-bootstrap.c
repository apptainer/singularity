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


int main(int argc, char ** argv) {
  if ( argv[1] == NULL ) {
    fprintf(stderr, "USAGE: UPDATE USAGE HERE!! SINGULARITY_IMAGE=[image] %s [command...]\n", argv[0]);
    return(1);
  }

  //For now copy code from image-mount.c
  singularity_priv_init();
  singularity_config_open(joinpath(SYSCONFDIR, "/singularity/singularity.conf"));
  singularity_sessiondir_init(containerimage);
  singularity_ns_user_unshare();
  singularity_ns_mnt_unshare();

  singularity_rootfs_init(containerimage);
  singularity_rootfs_mount();

  free(containerimage);

  singularity_message(VERBOSE, "Setting SINGULARITY_ROOTFS to '%s'\n", singularity_rootfs_dir());
  setenv("SINGULARITY_ROOTFS", singularity_rootfs_dir(), 1);

  if( singularity_fork() != 0 ) {
    singularity_message(DEBUG, "Parent process is exiting\n");
    return(-1);
  }
  
  mainsh();
  driverv2();
  prebootstrap();
  build-docker();
  postbootstrap();
  
}
