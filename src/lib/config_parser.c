/* 
 * Copyright (c) 2015-2016, Gregory M. Kurtzer. All rights reserved.
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

#define _GNU_SOURCE
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

FILE *config_fp = NULL;

int singularity_config_open(char *config_path) {
    singularity_message(VERBOSE, "Opening configuration file: %s\n", config_path);
    if ( is_file(config_path) == 0 ) {
        if ( ( config_fp = fopen(config_path, "re") ) != NULL ) { // Flawfinder: ignore (we have to open the file...)
            return(0);
        }
    }
    singularity_message(ERROR, "Could not open configuration file %s: %s\n", config_path, strerror(errno));
    return(-1);
}

void singularity_config_close(void) {
    singularity_message(VERBOSE, "Closing configuration file\n");
    if ( config_fp != NULL ) {
        fclose(config_fp);
        config_fp = NULL;
    }
}

void singularity_config_rewind(void) {
    singularity_message(DEBUG, "Rewinding configuration file\n");
    if ( config_fp != NULL ) {
        rewind(config_fp);
    }
}

char *singularity_config_get_value(char *key) {
    char *config_key;
    char *config_value;
    char *line;

    if ( config_fp == NULL ) {
        singularity_message(ERROR, "Called singularity_config_get_value() before opening a config!\n");
        ABORT(255);
    }

    line = (char *)malloc(MAX_LINE_LEN);

    singularity_message(DEBUG, "Called singularity_config_get_value(%s)\n", key);

    while ( fgets(line, MAX_LINE_LEN, config_fp) ) {
        if ( ( config_key = strtok(line, "=") ) != NULL ) {
            chomp(config_key);
            if ( strcmp(config_key, key) == 0 ) {
                if ( ( config_value = strdup(strtok(NULL, "=")) ) != NULL ) {
                    chomp(config_value);
                    singularity_message(VERBOSE2, "Got config key %s (= '%s')\n", key, config_value);
                    return(config_value);
                }
            }
        }
    }
    free(line);

    singularity_message(DEBUG, "No configuration file entry found for '%s'\n", key);
    return(NULL);
}


int singularity_config_get_bool(char *key, int def) {
    char *config_value;

    singularity_message(DEBUG, "Called singularity_config_get_bool(%s, %d)\n", key, def);

    if ( ( config_value = singularity_config_get_value(key) ) != NULL ) {
        if ( strcmp(config_value, "yes") == 0 ||
                strcmp(config_value, "y") == 0 ||
                strcmp(config_value, "1") == 0 ) {
            singularity_message(DEBUG, "Return singularity_config_get_bool(%s, %d) = 1\n", key, def);
            return(1);
        } else if ( strcmp(config_value, "no") == 0 ||
                strcmp(config_value, "n") == 0 ||
                strcmp(config_value, "0") == 0 ) {
            singularity_message(DEBUG, "Return singularity_config_get_bool(%s, %d) = 0\n", key, def);
            return(0);
        } else {
            singularity_message(ERROR, "Unsupported value for configuration boolean key '%s' = '%s'\n", key, config_value);
            singularity_message(ERROR, "Returning default value: %s\n", ( def == 1 ? "yes" : "no" ));
            ABORT(255);
        }
    } else {
        singularity_message(DEBUG, "Undefined configuration for '%s', returning default: %s\n", key, ( def == 1 ? "yes" : "no" ));
        return(def);
    }

    return(-1);
}
