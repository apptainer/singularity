/* 
 * Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
 *
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

    if ( key == NULL ) {
        singularity_message(ERROR, "Internal - Called keypair() with NULL key\n");
        ABORT(255);
    }

    hash_entry.key = strdup(key);

    if ( value == NULL ) {
        hash_entry.data = NULL;
    } else {
        hash_entry.data = strdup(value);
    }

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
            char *tok;
            char *string = strdup(*env++);

            if ( string == NULL ) {
                continue;
            } 

            if ( strncmp(string, "SINGULARITY_", 12) != 0 ) {
                continue;
            }

            tok = strchr(string, '=');
            *tok = '\0';

            string += 12; // Move string over so that SINGULARITY_ is skipped over

            singularity_registry_set(string, tok+1);
        }
    }
}


char *singularity_registry_get(char *key) {
    ENTRY *found;
    char *upperkey;
    int i = 0;
    int len = strlength(key, MAX_KEY_LEN);

    upperkey = (char *) malloc(len + 1);

    singularity_registry_init();

    for ( i = 0; i < len; ++i )
        upperkey[i] = toupper(key[i]);
    upperkey[len] = '\0';

    if ( hsearch_r(keypair(upperkey, NULL), FIND, &found, &htab) == 0 ) {
        singularity_message(DEBUG, "Returning NULL on '%s'\n", upperkey);
        return(NULL);
    }
    
    singularity_message(DEBUG, "Returning value from registry: '%s' = '%s'\n", upperkey, (char *)found->data);

    return(found->data ? (strdup(found->data)) : NULL);
}


int singularity_registry_set(char *key, char *value) {
    ENTRY *prev;
    char *upperkey;
    int i = 0;
    int len = strlength(key, MAX_KEY_LEN);

    upperkey = (char *) malloc(len + 1);

    singularity_registry_init();

    for ( i = 0; i < len; ++i )
        upperkey[i] = toupper(key[i]);
    upperkey[len] = '\0';

    singularity_message(VERBOSE2, "Adding value to registry: '%s' = '%s'\n", upperkey, value);

    if ( hsearch_r(keypair(upperkey, value), FIND, &prev, &htab) != 0 ) {
        singularity_message(VERBOSE2, "Found prior value for '%s', overriding with '%s'\n", key, value);
        prev->data = value ? strdup(value) : NULL;
    } else {
        if ( hsearch_r(keypair(upperkey, value), ENTER, &prev, &htab) == 0 ) {
            singularity_message(ERROR, "Internal error - Unable to set registry entry ('%s' = '%s'): %s\n", key, value, strerror(errno));
            ABORT(255);
        }
    }
    singularity_message(DEBUG, "Returning singularity_registry_set(%s, %s) = 0\n", key, value);

    return(0);
}



