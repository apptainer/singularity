/* 
 * Copyright (c) 2015-2016, Gregory M. Kurtzer. All rights reserved.
 * 
 * “Singularity” Copyright (c) 2016, The Regents of the University of California,
 * through Lawrence Berkeley National Laboratory (subject to receipt of any
 * required approvals from the U.S. Dept. of Energy).  All rights reserved.
 * 
 * If you have questions about your rights to use or distribute this software,
 * please contact Berkeley Lab's Innovation & Partnerships Office at
 * IPO@lbl.gov.
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



char *config_get_key_value(FILE *fp, char *key) {
    char *config_key;
    char *config_value;
    char line[1024];

    while ( fgets(line, sizeof(line), fp) ) {

        if ( ( config_key = strtok(line, "=") ) != NULL ) {
            chomp(config_key);
            if ( strcmp(config_key, key) == 0 ) {
                if ( ( config_value = strtok(NULL, "=") ) != NULL ) {
                    chomp(config_value);
                    if ( config_value[0] == ' ' ) {
                        config_value++;
                    }
                    return(config_value);
                }
            }
        }
    }

    return(NULL);
}


int config_get_key_bool(FILE *fp, char *key) {
    char *config_value;

    if ( ( config_value = config_get_key_value(fp, key) ) != NULL ) {
        if ( strcmp(config_value, "yes") == 0 ||
                strcmp(config_value, "y") == 0 ||
                strcmp(config_value, "1") == 0 ) {
            return(1);
        } else if ( strcmp(config_value, "no") == 0 ||
                strcmp(config_value, "n") == 0 ||
                strcmp(config_value, "0") == 0 ) {
            return(0);
        } else {
            fprintf(stderr, "ERROR: Unsupported value for configuration boolean key '%s' = '%s'\n", key, config_value);
            return(-1);
        }
    }

    return(0);
}
