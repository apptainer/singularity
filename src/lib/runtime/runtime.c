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
#include <sys/types.h>
#include <sys/mount.h>
#include <sys/wait.h>
#include <unistd.h>
#include <stdlib.h>
#include <sched.h>


#include "util/file.h"
#include "util/util.h"
#include "lib/message.h"
#include "./ns/ns.h"
#include "./mounts/mounts.h"
#include "./files/files.h"
#include "./enter/enter.h"


static char *container_directory = NULL;
static char *temp_directory = NULL;

char *singularity_runtime_containerdir(char *directory) {
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

char *singularity_runtime_tmpdir(char *directory) {
    if ( directory != NULL ) {
        if ( is_dir(directory) == 0 ) {
            temp_directory = strdup(directory);
        } else {
            singularity_message(ERROR, "Session path is not a directory: %s\n", directory);
            ABORT(255);
        }
    }

    return(temp_directory);
}



int singularity_runtime_ns(void) {
    return(_singularity_runtime_ns());
}

int singularity_runtime_mounts(void) {
    if ( singularity_runtime_containerdir(NULL) == NULL ) {
        singularity_message(ERROR, "The runtime container directory has not been set!\n");
        ABORT(5);
    }

    return(_singularity_runtime_mounts());
}

int singularity_runtime_files(void) {
    if ( singularity_runtime_containerdir(NULL) == NULL ) {
        singularity_message(ERROR, "The runtime container directory has not been set!\n");
        ABORT(5);
    }

    return(_singularity_runtime_files());
}

int singularity_runtime_enter(void) {
    if ( singularity_runtime_containerdir(NULL) == NULL ) {
        singularity_message(ERROR, "The runtime container directory has not been set!\n");
        ABORT(5);
    }

    return(_singularity_runtime_enter());
}

