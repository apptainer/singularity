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
#include "lib/bootstrap_parser.h"
#include "lib/singularity.h"


void singularity_bootstrap_init() {

  //sanity check here?? Or maybe just change name of _run_module to _init
  singularity_bootstrap_run_module();
}


void singularity_bootstrap_run_module() {
  char *module_name;

  singularity_bootdef_rewind();
  if ( ( module_name = singularity_bootdef_get_value("Bootstrap") ) == NULL ) {
    singularity_message(ERROR, "Bootstrap definition file does not contain a Bootstrap: line");
    ABORT(255);
    
  } else {
    singularity_message(INFO, "Bootstrap module %s found, running module builder\n", module_name);

    if ( strcmp(module_name, "docker") ) { //Docker
      singularity_bootstrap_docker_init();
      
    } else if ( strcmp(module_name, "yum") ) { //Yum
      singularity_bootstrap_yum_init();
      
    } else if ( strcmp(module_name, "debootstrap") ) { //Debootstrap
      singularity_bootstrap_debootstrap_init();
      
    } else if ( strcmp(module_name, "arch") ) { //Arch
      singularity_bootstrap_arch_init();
      
    } else if ( strcmp(module_name, "busybox") ) { //Busybox
      singularity_bootstrap_busybox_init();
      
    } else {
      singularity_message(ERROR, "Could not parse bootstrap module of type: %s", module_name);
      ABORT(255);
    }

}
