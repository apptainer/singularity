/* 
 * Copyright (c) 2016, Michael W. Bauer. All rights reserved.
 * 
 * “Singularity” Copyright (c) 2016, The Regents of the University of California,
 * through Lawrence Berkeley National Laboratory (subject to receipt of any
 * required approvals from the U.S. Dept. of Energy).  All rights reserved.
 * 
 * This software is licensed under a customized 3-clause BSD license.  Please
 * consult LICENSE file distributed with the sources of this project regarding
 * your rights to use or distribute this software.
 * 
 * NOTICE.  This Software was developed under funding from the U.S. Department of
 * Energy and the U.S. Government consequently retains certain rights. As such,
 * the U.S. Government has been granted for itself and others acting on its
 * behalf a paid-up, nonexclusive, irrevocable, worldwide license in the Software
 * to reproduce, distribute copies to the public, prepare derivative works, and
 * perform publicly and display publicly, and to permit other to do so. 
 * 
 */

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
#include "util/file.h"
#include "lib/message.h"


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
  singularity_message(ERROR, "Could not open bootstrap definition file %s: %s\n", bootdef_path, strerror(errno));
  return(-1);
}

void singularity_bootdef_rewind() {
  singularity_message(VERBOSE, "Rewinding bootstrap definition file\n");
  if ( bootdef_fp != NULL ) {
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

  while ( fgets(line, MAX_LINE_LEN, bootdef_fp) ) {
    if ( ( bootdef_key = strtok(line, ":") ) != NULL ) {
      chomp(bootdef_key);
      if ( strcmp(bootdef_key, key) == 0 ) {
	if ( ( bootdef_value = strdup(strtok(NULL, ":")) ) != NULL ) {
	  chomp(bootdef_value);
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

//Returns section args as well as leaves file open at first line of the script
//Returns NULL when section not found
char *singularity_bootdef_section_find(char *section_name) {
  char *line;
  char *tok;
  char *args;

  singularity_message(VERBOSE, "Searching for section %%%s\n", section_name);
  if ( bootdef_fp == NULL ) {
    singularity_message(ERROR, "Called singularity_bootdef_section_find() before opening a bootstrap definition file. Aborting...\n");
    ABORT(255);
  }

  singularity_bootdef_rewind();
  line = (char *)malloc(MAX_LINE_LEN);

  singularity_message(DEBUG, "Scanning file for start of %%%s section\n", section_name);
  while ( fgets(line, MAX_LINE_LEN, bootdef_fp) ) {
    chomp(line);

    if ( ( tok = strtok(line, "%% :") ) != NULL ) {
      singularity_message(DEBUG, "Comparing token: %s to section name: %s\n", tok, section_name);

      if ( strcmp(tok, section_name) == 0 ) {
	args = strdup("-e -x");
	
	while ( ( tok = strtok(NULL, "%% :") ) != NULL ) {
	  args = strjoin( args, strjoin(" ", tok) );
	}
	
	singularity_message(DEBUG, "Returning args: %s\n", args);
	free(line);
	return(args);
      }
    }
  }
  singularity_message(DEBUG, "Returning NULL\n");
  free(line);
  return(NULL);
}



//Can either directly call on get-section binary, or reimplement it here. Not sure what the best idea is?
char *singularity_bootdef_section_get(char **script, char *section_name) {
  char *script_args;
  char *line;
  int len = 1;

  singularity_message(VERBOSE, "Attempting to find and return script defined by section %%%s\n", section_name);
  if( ( script_args = singularity_bootdef_section_find(section_name) ) == NULL ) {
    singularity_message(DEBUG, "Unable to find section: %%%s in bootstrap definition file\n", section_name);
    return(NULL);
  }

  *script = strdup("");
  line = (char *)malloc(MAX_LINE_LEN);
  while ( fgets(line, MAX_LINE_LEN, bootdef_fp) ) {
    singularity_message(DEBUG, "Reading line: %s", line);
    if( strncmp(line, "%%", 1) == 0 ) {
      break;
    } else {
      chomp(line);
      *script = strjoin( *script, strjoin("\n", line) );
      singularity_message(DEBUG, "script: %s\n", *script);
      /*len = len + strlength(line, 2048);
      if ( ( *script = realloc( *script, len) ) == NULL ) {
	singularity_message(ERROR, "Unable to allocate enough memory. Aborting...\n");
	ABORT(255);
      }
      snprintf(*script, len, "%s%s", *script, line);*/
    }
  }
  free(line);
  return(script_args);
}
