#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <errno.h>
#include <string.h>
#include <fcntl.h>
#include <limits.h>

#include "config.h"
#include "util/util.h"
#include "lib/message.h"
#include "util/file.h"


#define MAX_LINE_LEN 2048

FILE *bootstrap_fp = NULL;

int singularity_bootstrap_open(char *bootstrap_path) {
  singularity_message(VERBOSE, "Opening bootstrap definition file: %s\n", bootstrap_path);
  if ( is_file(bootstrap_path) == 0 ) {
    if ( ( bootstrap_fp = fopen(bootstrap_path, "r") ) != NULL ) { // Flawfinder: ignore (we have to open the file...)
      return(0);
    }
  }
  singularity_message(ERROR, "Could not open bootstrap definition file %s: %s\n", bootstrap_path, strerror(errno));
  return(-1);
}

singularity_bootstrap_key_get()


singularity_bootstrap_keys_get()


singularity_bootstrap_section_exists()


singularity_bootstrap_section_args()


singularity_bootstrap_section_get()


singularity_bootstrap_parse_opts()
