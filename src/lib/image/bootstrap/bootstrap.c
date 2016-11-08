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

  //Check for $SINGULARITY_ROOTFS && $SINGULARITY_libexecdir definitions

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
    //singularity_priv_init(); //We need SUID to escalate privs for non-priv bootstrap, initialize that here and error out if we can't do it

    singularity_message(DEBUG, "Running bootstrap driver v2\n");

    singularity_prebootstrap_init(); //lib/bootstrap/prebootstrap/prebootstrap.c

    singularity_bootstrap_module_init(); //lib/bootstrap/bootstrap.c

    singularity_postbootstrap_init(); //lib/bootstrap/postbootstrap/postbootstrap.c

    singularity_bootdef_close();

  }


  return(0);
}
