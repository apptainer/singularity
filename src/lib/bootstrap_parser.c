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

FILE *bootdef_fp = NULL;

//All bootstrap definition file parser functions should follow singularity_bootdef_* naming convention

int singularity_bootdef_open(char *bootdef_path) {
  singularity_message(VERBOSE, "Opening bootstrap definition file: %s\n", bootdef_path);
  if ( is_file(bootdef_path) == 0 ) {
    if ( ( bootdef_fp = fopen(bootdef_path, "r") ) != NULL ) { // Flawfinder: ignore (we have to open the file...)
      return(0);
    }
  }
  singularity_message(ERROR, "Could not open bootstrap definition file %s: %s\n", bootstrap_path, strerror(errno));
  return(-1);
}

void singularity_bootdef_rewind() {
  singularity_message(DEBUG, "Rewinding bootstrap definition file\n");
  if ( _fp != NULL ) {
    rewind(bootdef_fp);
  }
}

void singularity_bootdef_close() {
  singularity_message(VERBOSE, "Closing bootstrap definition file\n");
  if ( bootdef_fp != NULL ) {
    fclose(bootdef_fp);
    bootdef_fp = NULL;
  }
}

//Equal to singularity_key_get in functions
char *singularity_bootdef_get_value(char *key) {
  char *bootdef_key;
  char *bootdef_value;
  char *line;

  if ( bootdef_fp == NULL ) {
    singularity_message(ERROR, "Called singularity_bootdef_get_value() before opening a bootstrap definition file!\n");
    ABORT(255);
  }

  line = (char *)malloc(MAX_LINE_LEN);

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

int singularity_bootdef_get_version() {
  char *v1_key = "DistType";

  if( singularity_bootdef_get_value(v1_key) != NULL ) {
    return(1);
  } else {
    return(2);
  }
}

singularity_bootdef_keys_get()


//returns section args as well as leaves file open at first line of the script
char *singularity_bootdef_section_find(char *section_name) {
  char *line;
  
  if ( bootdef_fp == NULL ) {
    singularity_message(ERROR, "Called singularity_bootdef_section_find() before opening a bootstrap definition file!\n");
    ABORT(255);
  }

  singularity_bootdef_rewind();
  line = (char *)malloc(MAX_LINE_LEN);
  
  while ( fgets(line, MAX_LINE_LEN, bootstrap_fp) ) {
    strtok(line, '%');
    if ( strcmp(strtok(NULL, " "), section_name) == 0 ) {
      return(line);
    }
  }
  return(NULL);
}



singularity_bootdef_section_args()


//Can either directly call on get-section binary, or reimplement it here. Not sure what the best idea is?
char *singularity_bootdef_section_get(char *script, char *section_name) {
  char *script_args;
  char *buf;
  if( ( script_args = singularity_bootdef_section_find(section_name) ) == NULL ) {
    singularity_message(DEBUG, "Unable to find section: %%%s in bootstrap definition file", section_name);
    return(NULL);
  }

  while ( fgets(line, MAX_LINE_LEN, bootstrap_fp) ) {
    if( strncmp(line, '%', 1) == 0 ) {
      break;
    } else {
      buf = script;
      sprintf(script, "%s%s", buf, line);
    }
  }
  return(script_args);
}


singularity_bootdef_parse_opts()
