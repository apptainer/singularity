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

#define _GNU_SOURCE
#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <string.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/mount.h>
#include <sys/wait.h>
#include <unistd.h>
#include <stdlib.h>
#include <sched.h>

#include "util/file.h"
#include "util/util.h"
#include "lib/message.h"
#include "lib/singularity.h"

/* Return 0 if successful, return -1 otherwise. */
int singularity_bootstrap_docker() {

    int index = 6;
    char ** python_args = malloc( sizeof(char *) * 9 );

    python_args[0] = strdup("python");
    python_args[1] = strdup(LIBEXECDIR "/singularity/python/cli.py");
    python_args[2] = strdup("--docker");
    python_args[3] = singularity_bootdef_get_value("From");
    python_args[4] = strdup("--rootfs");
    python_args[5] = singularity_rootfs_dir();

    if ( python_args[3] == NULL ) {
        singularity_message(VERBOSE, "Unable to bootstrap with docker container, missing From in definition file\n");
        return(1);
    }
  
    if ( ( python_args[index] = singularity_bootdef_get_value("IncludeCmd") ) != NULL ) {
        if ( strcmp(python_args[index], "yes") == 0 ) {
            python_args[index] = strdup("--cmd");
            index++;
        } else {
            python_args[index] = NULL;
        }
    }

    if ( ( python_args[index] = singularity_bootdef_get_value("Registry") ) != NULL ) {
        index++;
    }
    if ( ( python_args[index] = singularity_bootdef_get_value("Token" ) ) != NULL ) {
        index++;
    }
  
    python_args = realloc(python_args, (sizeof(char *) * index) ); //Realloc to free space at end of python_args, is this necessary?
    
    singularity_message(DEBUG, "\n 1: %s \n2: %s \n3: %s \n4: %s \n5: %s", python_args[1], python_args[2], python_args[3], python_args[4], python_args[5]);

    //Python libexecdir/singularity/python/cli.py --docker $docker_image --rootfs $rootfs $docker_cmd $docker_registry $docker_auth
    return(singularity_fork_exec(python_args));
}
