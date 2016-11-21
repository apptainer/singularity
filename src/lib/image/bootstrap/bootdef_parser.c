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

/* 
 * Opens up config file for reading. Config parsing works by scanning
 * the file line by line. bootdef_fp will not be reset to the beginning
 * of the file after each function, you must do this yourself. Otherwise
 * the next function call will pick up where the file was left from
 * the last function.
 *
 * @param char *bootdef_path pointer to string containing path to configuration file
 * @returns 0 on success, -1 on failure 
 */
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

/*  
 * Reset bootdef_fp to line 0
 *
 * @returns nothing
 */
void singularity_bootdef_rewind() {
  singularity_message(VERBOSE, "Rewinding bootstrap definition file\n");
  if ( bootdef_fp != NULL ) {
    rewind(bootdef_fp);
  }
}

/*
 * Closes bootdef_fp
 * 
 * @returns nothing
 */
void singularity_bootdef_close() {
  singularity_message(VERBOSE, "Closing bootstrap definition file\n");
  if ( bootdef_fp != NULL ) {
    fclose(bootdef_fp);
    bootdef_fp = NULL;
  }
}

/* 
 * Moves line by line through bootdef_fp until key is found. Once key is 
 * found the value is returned. The file remains opened at the line 
 * that contained key, thus requiring multiple calls to find all values
 * corresponding with key. Should call singularity_bootdef_rewind() before
 * searching for a new key to ensure entire config file is searched. 
 *
 * @param char *key pointer to string containing key to search for in bootdef_fp
 * @returns NULL if key not found, otherways returns value
 */
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

/*
 * Finds out whether bootdef_fp uses driver-v1 or driver-v2 syntax
 * 
 * @returns 1 if driver-v1, 2 if driver-v2
 */
int singularity_bootdef_get_version() {
  char *v1_key = "DistType";

  if( singularity_bootdef_get_value(v1_key) != NULL ) {
    return(1);
  } else {
    return(2);
  }
}

/*
 * Searches the bootdef file for the script section given by section_name. Leaves
 * the file pointer open to the first line of the script if found.
 *
 * @param char *section_name pointer to string containing name of script section to search for
 * @returns 0 if section was successfully located, -1 if it was not
 */
int singularity_bootdef_section_find(char *section_name) {
  char *line;
  char *tok;

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
	singularity_message(DEBUG, "Found %%%s section, returning 0.\n", section_name);
	free(line);
	return(0);
      }
    }
  }
  singularity_message(DEBUG, "Unable to find %%%s section\n", section_name);
  free(line);
  return(-1);
}

/*
 * Locates script defined by section_name in bootdef_fp, parsers each line
 * of the script and concatenates all the commands into one long string defined
 * at *script. Each command is stripped of leading/trailing whitespace, and each
 * is separated by a \n newline character
 *
 * @param char **script pointer to a pointer where the script should be stored
 * @param char *section_name pointer to string containing name of script section to parse
 * @returns 0 if script was found, -1 if script was not found
 */
int singularity_bootdef_section_get(char **script, char *section_name) {
  char *line;

  singularity_message(VERBOSE, "Attempting to find and return script defined by section %%%s\n", section_name);
  if( singularity_bootdef_section_find(section_name) == -1 ) {
    singularity_message(DEBUG, "Unable to find section: %%%s in bootstrap definition file\n", section_name);
    return(-1);
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
    }
  }
  free(line);
  return(0);
}
