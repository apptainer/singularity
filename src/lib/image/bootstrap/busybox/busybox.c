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

//Return 0 if successful, return 1 otherwise.
int singularity_bootstrap_busybox() {
  char ** module_script = malloc( sizeof(char *) * 1);
  module_script[0] = strdup(LIBEXECDIR "/singularity/bootstrap/modules-v2/build-busybox.sh");
  singularity_message(DEBUG, "Running %s bootstrapping script\n", module_script[0]);
  return(singularity_fork_exec(module_script));
}
