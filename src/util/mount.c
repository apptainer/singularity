/*
 * Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
 * 
 * See the COPYRIGHT.md file at the top-level directory of this distribution and at
 * https://github.com/singularityware/singularity/blob/master/COPYRIGHT.md.
 * 
 * This file is part of the Singularity Linux container project. It is subject to the license
 * terms in the LICENSE.md file found in the top-level directory of this distribution and
 * at https://github.com/singularityware/singularity/blob/master/LICENSE.md. No part
 * of Singularity, including this file, may be copied, modified, propagated, or distributed
 * except according to the terms contained in the LICENSE.md file.
 * 
*/

#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <string.h>
#include <sys/stat.h>
#include <sys/mount.h>
#include <sys/fsuid.h>
#include <unistd.h>
#include <stdlib.h>
#include <limits.h>

#include "config.h"
#include "util/file.h"
#include "util/util.h"
#include "util/message.h"
#include "util/privilege.h"

#define MAX_LINE_LEN 2048

int singularity_mount(const char *source, const char *target,
                      const char *filesystemtype, unsigned long mountflags,
                      const void *data) {
    if ( ( mountflags & MS_BIND ) ) {
        setfsuid(singularity_priv_getuid());
    }
    return mount(source, target, filesystemtype, mountflags, data);
}

int check_mounted(char *mountpoint) {
    int retval = -1;
    FILE *mounts;
    char *line = (char *)malloc(MAX_LINE_LEN);
    char *rootfs_dir = CONTAINER_FINALDIR;
    unsigned int mountpoint_len = strlength(mountpoint, PATH_MAX);
    char *real_mountpoint;

    singularity_message(DEBUG, "Opening /proc/mounts\n");
    if ( ( mounts = fopen("/proc/mounts", "r") ) == NULL ) { // Flawfinder: ignore
        singularity_message(ERROR, "Could not open /proc/mounts: %s\n", strerror(errno));
        ABORT(255);
    }

    if ( mountpoint[mountpoint_len-1] == '/' ) {
        singularity_message(DEBUG, "Removing trailing slash from string: %s\n", mountpoint);
        mountpoint[mountpoint_len-1] = '\0';
    }

    real_mountpoint = realpath(mountpoint, NULL); // Flawfinder: ignore
    if ( real_mountpoint == NULL ) {
        // mountpoint doesn't exists
        return(retval);
    }

    singularity_message(DEBUG, "Iterating through /proc/mounts\n");
    while ( fgets(line, MAX_LINE_LEN, mounts) != NULL ) {
        (void) strtok(strdup(line), " ");
        char *mount = strtok(NULL, " ");

        // Check to see if mountpoint is already mounted
        if ( strcmp(joinpath(rootfs_dir, real_mountpoint), mount) == 0 ) {
            singularity_message(DEBUG, "Mountpoint is already mounted: %s\n", mountpoint);
            retval = 1;
            break;
        }

        // Check to see if path is in container root
        if ( strncmp(rootfs_dir, mount, strlength(rootfs_dir, 1024)) != 0 ) {
            continue;
        }

        // Check to see if path is ot the container root
        if ( strcmp(mount, rootfs_dir) == 0 ) {
            continue;
        }
    }

    fclose(mounts);
    free(line);
    free(real_mountpoint);

    return(retval);
}

