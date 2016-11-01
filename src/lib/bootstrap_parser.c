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

//Should these functions be singularity_bootstrap_* or singularity_bootstrap_def_*?

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

void singularity_bootstrap_rewind() {
  singularity_message(DEBUG, "Rewinding bootstrap definition file\n");
  if ( _fp != NULL ) {
    rewind(bootstrap_fp);
  }
}

void singularity_bootstrap_close() {
  singularity_message(VERBOSE, "Closing bootstrap definition file\n");
  if ( bootstrap_fp != NULL ) {
    fclose(bootstrap_fp);
    bootstrap_fp = NULL;
  }
}

//Equal to singularity_key_get in functions
char *singularity_bootstrap_get_value(char *key) {
  char *bootdef_key;
  char *bootdef_value;
  char *line;

  if ( bootstrap_fp == NULL ) {
    singularity_message(ERROR, "Called singularity_bootstrap_get_value() before opening a bootstrap definition file!\n");
    ABORT(255);
  }

  line = (char *)mallaoc(MAX_LINE_LEN);

  while ( fgets(line, MAX_LINE_LEN, bootstrap_fp) ) {
    if ( ( bootdef_key = strtok(line, ":") ) != NULL ) {
      chomp(bootdef_key);
      if ( strcmp(bootdef_key, key) == 0 ) {
	if ( ( bootdef_value = strdup(strtok(NULL, ":")) ) != NULL ) {
	  chomp(bootdef_value);
	  if ( bootdef_value[0] == ' ' ) {
	    bootdef_value++;
	  }
	  singularity_message(VERBOSE2, "Got bootstrap definition key %s(: '%s')\n", key, bootdef_value);
	  return(bootdef_value);
	}
      }
    }
  }
  free(line);

  singularity_message(DEBUG, "No bootstrap definition file entry found for '%s'\n", key);
  return(NULL);
}

singularity_bootstrap_keys_get()



singularity_bootstrap_section_exists()



singularity_bootstrap_section_args()


//Can either directly call on get-section binary, or reimplement it here. Not sure what the best idea is?
singularity_bootstrap_section_get()


singularity_bootstrap_parse_opts()
