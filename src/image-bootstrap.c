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

  //Check for $SINGULARITY_ROOTFS && $SINGULARITY_libexecdir definitions
  
  //image-mount has finished, we are now inside a fork of image-mount running image-bootstrap binary instead of bootstrap.sh
  
  //Parse args for bootstrap_version

  //mkdir -d /tmp/singularity-bootstrap.XXXXXXX

  singularity_prebootstrap(); //lib/bootstrap/prebootstrap/prebootstrap.c
  
  singularity_bootstrap_init(); //lib/bootstrap/bootstrap.c

  singularity_postbootstrap(); //lib/bootstrap/postbootstrap/postbootstrap.c

  return(0);
}
