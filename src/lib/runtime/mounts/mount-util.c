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

#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <string.h>
#include <sys/stat.h>
#include <unistd.h>
#include <stdlib.h>
#include <limits.h>

#include "util/file.h"
#include "util/util.h"
#include "util/message.h"

#include "../runtime.h"

#define MAX_LINE_LEN 2048

int check_mounted(char *mountpoint) {
    int retval = -1;
    FILE *mounts;
    char *line = (char *)malloc(MAX_LINE_LEN);
    char *rootfs_dir = singularity_runtime_rootfs(NULL);
    unsigned int mountpoint_len = strlength(mountpoint, PATH_MAX);

    singularity_message(DEBUG, "Opening /proc/mounts\n");
    if ( ( mounts = fopen("/proc/mounts", "r") ) == NULL ) { // Flawfinder: ignore
        singularity_message(ERROR, "Could not open /proc/mounts: %s\n", strerror(errno));
        ABORT(255);
    }

    if ( mountpoint[mountpoint_len-1] == '/' ) {
        singularity_message(DEBUG, "Removing trailing slash from string: %s\n", mountpoint);
        mountpoint[mountpoint_len-1] = '\0';
    }

    singularity_message(DEBUG, "Iterating through /proc/mounts\n");
    while ( fgets(line, MAX_LINE_LEN, mounts) != NULL ) {
        (void) strtok(strdup(line), " ");
        char *mount = strtok(NULL, " ");

        // Check to see if path is in container root
        if ( strncmp(rootfs_dir, mount, strlength(rootfs_dir, 1024)) != 0 ) {
            continue;
        }

        // Check to see if path is ot the container root
        if ( strcmp(mount, rootfs_dir) == 0 ) {
            continue;
        }

        // Check to see if mountpoint is already mounted
        if ( strcmp(joinpath(rootfs_dir, mountpoint), mount) == 0 ) {
            singularity_message(DEBUG, "Mountpoint is already mounted: %s\n", mountpoint);
            retval = 1;
            break;
        }
    }

    fclose(mounts);
    free(line);

    return(retval);
}
