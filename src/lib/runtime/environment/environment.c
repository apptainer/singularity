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


extern char **environ;

int _singularity_runtime_environment(void) {
    int retval = 0;
    int cleanall = 0;
    char **env = environ;
    char **envclone;
    int i;
    int envlen = 0;

    if ( singularity_registry_get("CLEANENV") != NULL ) {
        cleanall = 1;
    }

    for(i = 0; env[i] != 0; i++) {
        envlen++;
    }

    envclone = (char**) malloc(i * sizeof(char *));

    for(i = 0; env[i] != 0; i++) {
        envclone[i] = strdup(env[i]);
    }

    for(i = 0; i < envlen; i++) {
        char *tok, *key;
        
        key = strtok_r(envclone[i], "=", &tok);

        if ( cleanall == 1 ) {
            singularity_message(DEBUG, "Unsetting environment variable: %s\n", key);
            unsetenv(key);
        } else {
            if ( strncmp(key, "SINGULARITY_", 12) == 0 ) {
                singularity_message(DEBUG, "Unsetting environment variable: %s\n", key);
                unsetenv(key);
            } else {
                singularity_message(DEBUG, "Leaving environment variable set: %s\n", key);
            }
        }
    }

    return(retval);
}

