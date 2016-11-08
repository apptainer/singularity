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
