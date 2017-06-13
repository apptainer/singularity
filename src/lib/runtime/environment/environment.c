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
#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <string.h>
#include <sys/stat.h>
#include <unistd.h>
#include <stdlib.h>

#include "util/file.h"
#include "util/util.h"
#include "util/message.h"
#include "util/privilege.h"
#include "util/registry.h"



int _singularity_runtime_environment(void) {
    char **current_env = environ;
    char **envclone;
    int envlen = 0;
    int i;


    // Copy and cache environment
    singularity_message(DEBUG, "Cloning environment\n");
    for(envlen = 0; current_env[envlen] != 0; envlen++) { }
    singularity_message(DEBUG, "Counted %d environment elements\n", envlen);
    envclone = (char**) malloc(envlen * sizeof(char *));
    for(i = 0; i < envlen; i++) {
        envclone[i] = strdup(current_env[i]);
    }


    // Clean environment
    if ( singularity_registry_get("CLEANENV") != NULL ) {
        char *term = envar_get("TERM", "_-.", 128);
        char *home = envar_path("HOME");

        singularity_message(DEBUG, "Sanitizing environment\n");
        if ( envclean() != 0 ) {
            singularity_message(ERROR, "Failed sanitizing environment\n");
            ABORT(255);
        }

        envar_set("LANG", "C", 1);
        envar_set("TERM", term, 1);
        envar_set("HOME", home, 1);
    } else {
        singularity_message(DEBUG, "Cleaning environment\n");
        for(i = 0; i < envlen; i++) {
            singularity_message(DEBUG, "Evaluating envar to clean: %s\n", envclone[i]);
            if ( strncmp(envclone[i], "SINGULARITY_", 12) == 0 ) {
                char *key, *tok;

                key = strtok_r(envclone[i], "=", &tok);

                singularity_message(DEBUG, "Unsetting environment variable: %s\n", key);
                unsetenv(key);
            }
        }
    }


    // Transpose environment
    singularity_message(DEBUG, "Transposing environment\n");
    for(i = 0; i < envlen; i++) {
        if ( strncmp(envclone[i], "SINGULARITYENV_", 15) == 0 ) {
            char *tok, *key, *val;
        
            key = strtok_r(envclone[i], "=", &tok);
            val = strtok_r(NULL, "\n", &tok);

            singularity_message(DEBUG, "Converting envar '%s' to '%s' = '%s'\n", key, &key[15], val);
            envar_set(&key[15], val, 1);
            unsetenv(key);
        }
    }

    for(i = 0; i < envlen; i++) {
        free(envclone[i]);
    }

    return(0);
}

