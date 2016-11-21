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

//Attempt to rebuild database after running OS installation, should fix issue @http://singularity.lbl.gov/bootstrap-image#bootstrap
//Return 0 if successful, return 1 otherwise.
int singularity_bootstrap_debootstrap() {
  char ** module_script = malloc( sizeof(char *) * 1);
  module_script[0] = strdup(LIBEXECDIR "/singularity/bootstrap/modules-v2/build-debootstrap.sh");
  return(singularity_fork_exec(module_script));
}
