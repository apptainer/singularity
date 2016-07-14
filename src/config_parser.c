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


#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <errno.h> 
#include <string.h>
#include <fcntl.h>  
#include <linux/limits.h>

#include "config.h"
#include "util.h"
#include "message.h"


#define MAX_LINE_LEN 2048


char *config_get_key_value(FILE *fp, char *key) {
    char *config_key;
    char *config_value;
    char *line;

    line = (char *)malloc(MAX_LINE_LEN);

    message(DEBUG, "Called config_get_key_value(fp, %s)\n", key);

    while ( fgets(line, MAX_LINE_LEN, fp) ) {
        if ( ( config_key = strtok(line, "=") ) != NULL ) {
            chomp(config_key);
            if ( strcmp(config_key, key) == 0 ) {
                if ( ( config_value = strdup(strtok(NULL, "=")) ) != NULL ) {
                    chomp(config_value);
                    if ( config_value[0] == ' ' ) {
                        config_value++;
                    }
                    message(DEBUG, "Return config_get_key_value(fp, %s) = %s\n", key, config_value);
                    return(config_value);
                }
            }
        }
    }
    free(line);

    message(DEBUG, "Return config_get_key_value(fp, %s) = NULL\n", key);
    return(NULL);
}


int config_get_key_bool(FILE *fp, char *key, int def) {
    char *config_value;

    message(DEBUG, "Called config_get_key_bool(fp, %s, %d)\n", key, def);

    if ( ( config_value = config_get_key_value(fp, key) ) != NULL ) {
        if ( strcmp(config_value, "yes") == 0 ||
                strcmp(config_value, "y") == 0 ||
                strcmp(config_value, "1") == 0 ) {
            message(DEBUG, "Return config_get_key_bool(fp, %s, %d) = 1\n", key, def);
            return(1);
        } else if ( strcmp(config_value, "no") == 0 ||
                strcmp(config_value, "n") == 0 ||
                strcmp(config_value, "0") == 0 ) {
            message(DEBUG, "Return config_get_key_bool(fp, %s, %d) = 0\n", key, def);
            return(0);
        } else {
            message(ERROR, "Unsupported value for configuration boolean key '%s' = '%s'\n", key, config_value);
            message(DEBUG, "Return config_get_key_bool(fp, %s, %d) = -1\n", key, def);
            return(-1);
        }
    }

    message(DEBUG, "Return config_get_key_bool(fp, %s, %d) = %d (DEFAULT)\n", key, def, def);
    return(def);
}
