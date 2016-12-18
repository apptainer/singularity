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
#include "util/message.h"
#include "./ns/ns.h"
#include "./mount/mount.h"
#include "./files/files.h"


char *container_directory = NULL;
unsigned int runtime_flags = 0;

char *singularity_runtime_dir(char *directory) {
    if ( directory != NULL ) {
        if ( is_dir(directory) == 0 ) {
            container_directory = strdup(directory);
        } else {
            singularity_message(ERROR, "Container path is not a directory: %s\n", directory);
            ABORT(255);
        }
    }

    return(container_directory);
}


int singularity_runtime_flags(unsigned int flags) {
    if ( flags > 0 ) {
        runtime_flags |= flags;
    }

    return(runtime_flags);
}


int singularity_runtime_check(void) {
    int retval = 0;

    if ( singularity_runtime_dir(NULL) == NULL ) {
        singularity_message(ERROR, "The runtime container directory has not been set!\n");
        ABORT(5);
    }

    singularity_message(VERBOSE, "Checking all runtime components\n");
    retval += singularity_runtime_ns_check();
    retval += singularity_runtime_mount_check();
    retval += singularity_runtime_files_check();
    
    return(retval);
}


int singularity_runtime_prepare(void) {
    int retval = 0;

    if ( singularity_runtime_dir(NULL) == NULL ) {
        singularity_message(ERROR, "The runtime container directory has not been set!\n");
        ABORT(5);
    }

    singularity_message(VERBOSE, "Preparing all runtime components\n");
    retval += singularity_runtime_ns_setup();
    retval += singularity_runtime_mount_setup();
    retval += singularity_runtime_files_setup();
    
    return(retval);
}


int singularity_runtime_activate(void) {
    int retval = 0;

    if ( singularity_runtime_dir(NULL) == NULL ) {
        singularity_message(ERROR, "The runtime container directory has not been set!\n");
        ABORT(5);
    }

    singularity_message(VERBOSE, "Activating all runtime components\n");
    retval += singularity_runtime_ns_do();
    retval += singularity_runtime_mount_do();
    retval += singularity_runtime_files_do();
    
    return(retval);
}
