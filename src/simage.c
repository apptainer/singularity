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
    fprintf(stderr, "USAGE: simage command args\n");
    return(1);
    
  } else {

    //Loop through argv, each time chopping off argv[0], until argv[1] is a relevant shell script or is empty
    singularity_priv_init(); //Make sure user is running as root before we add SUID code
    while ( true ) {
      singularity_message(DEBUG, "Running %s %s workflow\n", argv[0], argv[1]);

      singularity_priv_init();
      if ( argv[1] == NULL ) {
	singularity_message(DEBUG, "Finished running simage command and returning\n");
	return(0);

      } else if ( strcmp(argv[1], "mount") == 0 ) {
	if ( singularity_image_mount(argc - 1, &argv[1]) != 0 ) {
	  singularity_priv_drop_perm();
	  return(1);
	}
	
      } else if ( strcmp(argv[1], "bind") == 0 ) {
	if ( singularity_image_bind(argc - 1, &argv[1]) != 0 ) {
	  singularity_priv_drop_perm();
	  return(1);
	}
	
      } else if ( strcmp(argv[1], "create") == 0 ) {
	if ( singularity_image_create(argc - 1, &argv[1]) != 0 ) {
	  singularity_priv_drop_perm();
	  return(1);
	}
	
      } else if ( strcmp(argv[1], "expand") == 0 ) {
	if ( singularity_image_expand(argc - 1, &argv[1]) != 0 ) {
	  singularity_priv_drop_perm();
	  return(1);
	}
	
      } else {
	singularity_priv_drop_perm(); //Drop all privs permanently and return to calling user
	return(singularity_fork_exec(&argv[1])); //Can NOT run this with root privs
      }
      
      argv++;
      argc--;
      singularity_priv_drop_perm();
    }
  }
}
