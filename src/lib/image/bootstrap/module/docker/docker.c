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
int singularity_bootstrap_docker_init() {

  int index = 6;
  char ** python_args = malloc( sizeof(char *) * 9 );

  python_args[0] = strdup("python");
  python_args[1] = strdup(LIBEXECDIR "/singularity/python/cli.py");
  python_args[2] = strdup("--docker");
  python_args[3] = singularity_bootdef_get_value("From");
  python_args[4] = strdup("--rootfs");
  python_args[5] = singularity_rootfs_dir();

  if ( python_args[3] == NULL ) {
    singularity_message(VERBOSE, "Unable to bootstrap with docker container, missing From in definition file\n");
    return(1);
  }
  
  if ( ( python_args[index] = singularity_bootdef_get_value("IncludeCmd") ) != NULL ) {
    index++;
  }

  if ( ( python_args[index] = singularity_bootdef_get_value("Registry") ) != NULL ) {
    index++;
  }

  if ( ( python_args[index] = singularity_bootdef_get_value("Token" ) ) != NULL ) {
    index++;
  }
  python_args = realloc(python_args, (sizeof(char *) * index) ); //Realloc to free space at end of python_args, is this necessary?
    
  
  
  //  python_args = {
  //  strdup("python"),
  //  strdup(LIBEXECDIR "/singularity/python/cli.py"),
  //  strdup("--docker"),
  //  singularity_bootdef_get_value("From"),
  //  strdup("--rootfs"),
  //  singularity_rootfs_dir()
  // }
  
  //Python libexecdir/singularity/python/cli.py --docker $docker_image --rootfs $rootfs $docker_cmd $docker_registry $docker_auth
  return(singularity_fork_exec(python_args));
}
