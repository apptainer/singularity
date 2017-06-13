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
#include <sys/types.h>
#include <sys/mount.h>
#include <sys/wait.h>
#include <unistd.h>
#include <stdlib.h>
#include <sched.h>

#include "util/file.h"
#include "util/util.h"
#include "util/message.h"
#include "util/privilege.h"
#include "util/config_parser.h"

#include "./ns/ns.h"
#include "./mounts/mounts.h"
#include "./files/files.h"
#include "./enter/enter.h"
#include "./overlayfs/overlayfs.h"
#include "./environment/environment.h"

#ifndef LOCALSTATEDIR
#error LOCALSTATEDIR not defined
#endif

static char *container_directory = NULL;

char *singularity_runtime_rootfs(char *directory) {
    if ( directory != NULL ) {
        if ( is_dir(directory) == 0 ) {
            singularity_message(DEBUG, "Setting container_directory = '%s'\n", directory);
            container_directory = strdup(directory);
        } else {
            singularity_message(ERROR, "Container path is not a directory: %s\n", directory);
            ABORT(255);
        }
    } else if ( container_directory == NULL ) {
        container_directory = joinpath(LOCALSTATEDIR, "/singularity/mnt/container");

        singularity_message(VERBOSE, "Set container directory to: %s\n", container_directory);

        singularity_message(DEBUG, "Checking for container directory\n");
        if ( is_dir(container_directory) != 0 ) {
            singularity_message(ERROR, "Container directory does not exist: %s\n", container_directory);
            ABORT(255);
        }

    }

    singularity_message(DEBUG, "Returning container_directory: %s\n", container_directory);
    return(container_directory);
}

int singularity_runtime_ns(unsigned int flags) {
    return(_singularity_runtime_ns(flags));
}

int singularity_runtime_overlayfs(void) {
    /* If a daemon already exists, skip this function */
    if( singularity_registry_get("DAEMON") == 1 )
        return(0);

    return(_singularity_runtime_overlayfs());
}

int singularity_runtime_environment(void) {
    return(_singularity_runtime_environment());
}

int singularity_runtime_mounts(void) {
    /* If a daemon already exists, skip this function */
    if( singularity_registry_get("DAEMON") == 1 )
        return(0);

    if ( singularity_runtime_rootfs(NULL) == NULL ) {
        singularity_message(ERROR, "The runtime container directory has not been set!\n");
        ABORT(5);
    }

    return(_singularity_runtime_mounts());
}

int singularity_runtime_files(void) {
    /* If a daemon already exists, skip this function */
    if( singularity_registry_get("DAEMON") == 1 )
        return(0);

    if ( singularity_runtime_rootfs(NULL) == NULL ) {
        singularity_message(ERROR, "The runtime container directory has not been set!\n");
        ABORT(5);
    }

    return(_singularity_runtime_files());
}

int singularity_runtime_enter(void) {
    if ( singularity_runtime_rootfs(NULL) == NULL ) {
        singularity_message(ERROR, "The runtime container directory has not been set!\n");
        ABORT(5);
    }

    return(_singularity_runtime_enter());
}

