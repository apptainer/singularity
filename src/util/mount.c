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
    uid_t fsuid = 0;

    if ( ( mountflags & MS_BIND ) ) {
        fsuid = singularity_priv_getuid();
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
    ret = mount(source, target, filesystemtype, mountflags, data);
    if ( singularity_priv_userns_enabled() == 0 && seteuid(singularity_priv_getuid()) < 0 ) {
        singularity_message(ERROR, "Failed to drop privileges: %s\n", strerror(errno));
        ABORT(255);
    }

    return ret;
}

int check_mounted(char *mountpoint) {
    int retval = -1;
    FILE *mounts;
    char *line = (char *)malloc(MAX_LINE_LEN);
    char *rootfs_dir = CONTAINER_FINALDIR;
    unsigned int mountpoint_len = strlength(mountpoint, PATH_MAX);
    char *real_mountpoint = strdup(mountpoint);

    singularity_message(DEBUG, "Checking if currently mounted: %s\n", mountpoint);

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

        char *test_mountpoint = strdup(real_mountpoint);

        while ( strcmp(test_mountpoint, "/") != 0 ) {
            char *full_test_path = NULL;
            char *tmp_test_path = joinpath(rootfs_dir, test_mountpoint);

            if ( is_link(tmp_test_path) == 0 ) {
                char *linktarget = realpath(tmp_test_path, NULL);
                if ( linktarget == NULL ) {
                    singularity_message(ERROR, "Could not identify the source of contained link: %s\n", test_mountpoint);
                    ABORT(255);
                }
                full_test_path = joinpath(rootfs_dir, linktarget);
                singularity_message(DEBUG, "parent directory is a link, resolved: %s->%s\n", joinpath(rootfs_dir, test_mountpoint), full_test_path);
                if ( strcmp(linktarget, "/") == 0 ) {
                    retval = 1;
                    free(test_mountpoint);
                    goto DONE;
                }
            } else {
                full_test_path = tmp_test_path;
            }

            // Check to see if mountpoint is already mounted
            if ( strcmp(full_test_path, mount) == 0 ) {
                singularity_message(DEBUG, "Mountpoint is already mounted: %s\n", test_mountpoint);
                retval = 1;
                free(test_mountpoint);
                goto DONE;
            }
            test_mountpoint = dirname(test_mountpoint);

        }

        free(test_mountpoint);

        // Check to see if path is in container root
        if ( strncmp(rootfs_dir, mount, strlength(rootfs_dir, 1024)) != 0 ) {
            continue;
        }

        // Check to see if path is ot the container root
        if ( strcmp(mount, rootfs_dir) == 0 ) {
            continue;
        }
    }

    DONE:
    fclose(mounts);
    free(line);
    free(real_mountpoint);

    return(retval);
}

