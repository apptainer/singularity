/*
 * Copyright (c) 2017-2018, SyLabs, Inc. All rights reserved.
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

#define _GNU_SOURCE
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
#include <libgen.h>

#include "config.h"
#include "util/file.h"
#include "util/util.h"
#include "util/message.h"
#include "util/privilege.h"

#define MAX_LINE_LEN 2048

int singularity_mount(const char *source, const char *target,
                      const char *filesystemtype, unsigned long mountflags,
                      const void *data) {
    int ret;
    int mount_errno;
    uid_t fsuid = 0;
    char dest[PATH_MAX];
    char *realdest;
    int target_fd = open(target, O_RDONLY);

    if ( target_fd < 0 ) {
        singularity_message(ERROR, "Target %s doesn't exist\n", target);
        ABORT(255);
    }

    if ( snprintf(dest, PATH_MAX-1, "/proc/self/fd/%d", target_fd) < 0 ) {
        singularity_message(ERROR, "Failed to determine path for target file descriptor\n");
        ABORT(255);
    }

    if ( ( mountflags & MS_BIND ) ) {
        fsuid = singularity_priv_getuid();
    }

    realdest = realpath(dest, NULL); // Flawfinder: ignore
    if ( realdest == NULL ) {
        singularity_message(ERROR, "Failed to get real path of %s %s\n", target, dest);
        ABORT(255);
    }

    if ( (mountflags & MS_PRIVATE) == 0 && (mountflags & MS_SLAVE) == 0 ) {
        if ( strncmp(realdest, CONTAINER_MOUNTDIR, strlen(CONTAINER_MOUNTDIR)) != 0 &&
             strncmp(realdest, CONTAINER_FINALDIR, strlen(CONTAINER_FINALDIR)) != 0 &&
             strncmp(realdest, CONTAINER_OVERLAY, strlen(CONTAINER_OVERLAY)) != 0 &&
             strncmp(realdest, SESSIONDIR, strlen(SESSIONDIR)) != 0 ) {
            singularity_message(VERBOSE, "Ignored, try to mount %s outside of container %s\n", target, realdest);
            free(realdest);
            close(target_fd);
            return(0);
        }
    }

    /* don't modify user groups */
    if ( singularity_priv_userns_enabled() == 0 ) {
        if ( seteuid(0) < 0 ) {
            singularity_message(ERROR, "Failed to escalate privileges: %s\n", strerror(errno));
            ABORT(255);
        }
        /* NFS root_squash option set uid 0 to nobody, force use of real user ID */
        setfsuid(fsuid);
    }

    ret = mount(source, dest, filesystemtype, mountflags, data);
    mount_errno = errno;

    close(target_fd);
    free(realdest);

    if ( singularity_priv_userns_enabled() == 0 && seteuid(singularity_priv_getuid()) < 0 ) {
        singularity_message(ERROR, "Failed to drop privileges: %s\n", strerror(errno));
        ABORT(255);
    }

    errno = mount_errno;
    return ret;
}

int check_mounted(char *mountpoint) {
    int retval = -1;
    FILE *mounts;
    char *line = (char *)malloc(MAX_LINE_LEN);
    char *rootfs_dir = CONTAINER_FINALDIR;
    unsigned int mountpoint_len = strlength(mountpoint, PATH_MAX);
    char *real_mountpoint;
    char procmounts[PATH_MAX];

    singularity_message(DEBUG, "Opening /proc/mounts\n");
    if ( ( mounts = fopen("/proc/mounts", "r") ) == NULL ) { // Flawfinder: ignore
        singularity_message(ERROR, "Could not open /proc/mounts: %s\n", strerror(errno));
        ABORT(255);
    }

    if ( mountpoint[mountpoint_len-1] == '/' ) {
        singularity_message(DEBUG, "Removing trailing slash from string: %s\n", mountpoint);
        mountpoint[mountpoint_len-1] = '\0';
    }

    real_mountpoint = realpath(joinpath(rootfs_dir, mountpoint), NULL); // Flawfinder: ignore
    if ( real_mountpoint == NULL ) {
        // mountpoint doesn't exists
        return(retval);
    }

    if ( snprintf(procmounts, PATH_MAX-1, "%s/proc/%d/mounts", rootfs_dir, getpid()) < 0 ) {
        singularity_message(ERROR, "Can't construct path %s/proc/%d/mounts\n", rootfs_dir, getpid());
        ABORT(255);
    }

    if ( strcmp(real_mountpoint, procmounts) == 0 ) {
        singularity_message(ERROR, "Attempt to override /proc/mounts, aborting\n");
        ABORT(255);
    }

    singularity_message(DEBUG, "Iterating through /proc/mounts\n");
    while ( fgets(line, MAX_LINE_LEN, mounts) != NULL ) {
        (void) strtok(strdup(line), " ");
        char *mount = strtok(NULL, " ");

        // Check to see if mountpoint is already mounted
        if ( strcmp(real_mountpoint, mount) == 0 ) {
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

