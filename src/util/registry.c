/* 
 * Copyright (c) 2015-2017, Gregory M. Kurtzer. All rights reserved.
 * 
 * Copyright (c) 2016-2017, The Regents of the University of California,
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
#include <ctype.h>
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <errno.h> 
#include <string.h>
#include <fcntl.h>  
#include <limits.h>
#include <search.h>
#include <glob.h>

#include "config.h"
#include "util/util.h"
#include "util/message.h"
#include "util/file.h"
#include "registry.h"


static int registry_initialized = 0;
static struct hsearch_data htab;
extern char **environ;


static ENTRY keypair(char *key, char *value) {
    ENTRY hash_entry;
    hash_entry.key = key;
    hash_entry.data = value;
    return hash_entry;
}


void singularity_registry_init(void) {
    if ( registry_initialized != 1 ) {
        char **env = environ;
        singularity_message(VERBOSE, "Initializing Singularity Registry\n");
        if ( hcreate_r(REGISTRY_SIZE, &htab) == 0 ) {
            singularity_message(ERROR, "Internal error - Unable to initalize registry core: %s\n", strerror(errno));
            ABORT(255);
        }

        registry_initialized = 1;

        while (*env) {
            char *tok, *key, *val;
            char *string = *env++;

            if ( strncmp(string, "SINGULARITY_", 12) != 0 ) {
                continue;
            }

            key = strtok_r(string, "=", &tok);
            val = strtok_r(NULL, "=", &tok);

            if ( key == NULL ) {
                continue;
            } 

            if ( val == NULL ) {
                val = "";
            }

            singularity_registry_set(&key[12], val);
        }
    }
}


char *singularity_registry_get(char *key) {
    ENTRY *found;
    char *upperkey;
    int i = 0;
    int len = strlength(key, MAX_KEY_LEN);

    upperkey = (char *) malloc(len);

    singularity_registry_init();

    while ( i <= len ) {
        upperkey[i] = toupper(key[i]);
        i++;
    }

    if ( hsearch_r(keypair(upperkey, NULL), FIND, &found, &htab) == 0 ) {
        return(NULL);
    }

    return(found->data);
}


int singularity_registry_set(char *key, char *value) {
    ENTRY *prev;
    char *upperkey;
    int i = 0;
    int len = strlength(key, MAX_KEY_LEN);

    upperkey = (char *) malloc(len);

    singularity_registry_init();

    while ( i <= len ) {
        upperkey[i] = toupper(key[i]);
        i++;
    }

    singularity_message(VERBOSE2, "Adding value to registry: '%s' = '%s'\n", upperkey, value);

    if ( singularity_registry_get(upperkey) != NULL ) {
        singularity_message(VERBOSE2, "Found prior value for '%s', overriding with '%s'\n", key, value);
        prev->data = value;
    } else {
        if ( hsearch_r(keypair(upperkey, value), ENTER, &prev, &htab) == 0 ) {
            singularity_message(ERROR, "Internal error - Unable to set registry entry ('%s' = '%s'): %s\n", key, value, strerror(errno));
            ABORT(255);
        }
    }

    return(0);
}



