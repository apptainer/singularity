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

  char *bootdef_path;
  
  if ( argv[1] == NULL ) {
    fprintf(stderr, "USAGE: UPDATE USAGE HERE!! SINGULARITY_IMAGE=[image] %s [command...]\n", argv[0]);
    return(1);
  }

  //Abort if we can't open the bootstrap definition file
  if( singularity_bootdef_open(argv[1]) != 0 ) {
    ABORT(255);
  }

  //Check for $SINGULARITY_ROOTFS && $SINGULARITY_libexecdir definitions
  
  //image-mount has finished, we are now inside a fork of image-mount running image-bootstrap binary instead of bootstrap.sh

  //Execute old driver-v1.sh if bootdef_ver = 1, else execute new bootstrap code
  if( ( bootstrap_ver = singularity_bootdef_get_version() ) == 1 ) {
    singularity_message(DEBUG, "Running bootstrap driver v1\n");

    singularity_bootdef_close();
    
    //Maybe directly use driver-v2.sh since it is outdated and we don't need to rewrite it for future use?
  }
  else {
    singularity_message(DEBUG, "Running bootstrap driver v2\n");

    singularity_prebootstrap(); //lib/bootstrap/prebootstrap/prebootstrap.c

    singularity_bootstrap_init(); //lib/bootstrap/bootstrap.c

    singularity_postbootstrap(); //lib/bootstrap/postbootstrap/postbootstrap.c

    singularity_bootdef_close();
    
  }
  
  
  //mkdir -d /tmp/singularity-bootstrap.XXXXXXX

    return(0);
}
