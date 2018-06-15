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

struct resolved_container_path {
    char *mountdir;
    char *finaldir;
    char *overlay;
    char *session;
};

static void resolve_container_path(struct resolved_container_path *container_path) {
    if ( container_path->mountdir == NULL ) {
        container_path->mountdir = realpath(CONTAINER_MOUNTDIR, NULL); // Flawfinder: ignore
        if ( container_path->mountdir == NULL ) {
            singularity_message(ERROR, "Failed to resolve path to %s: %s\n", CONTAINER_MOUNTDIR, strerror(errno));
            ABORT(255);
        }
    }
    if ( container_path->finaldir == NULL ) {
        container_path->finaldir = realpath(CONTAINER_FINALDIR, NULL); // Flawfinder: ignore
        if ( container_path->finaldir == NULL ) {
            singularity_message(ERROR, "Failed to resolve path to %s: %s\n", CONTAINER_FINALDIR, strerror(errno));
            ABORT(255);
        }
    }
    if ( container_path->overlay == NULL ) {
        container_path->overlay = realpath(CONTAINER_OVERLAY, NULL); // Flawfinder: ignore
        if ( container_path->overlay == NULL ) {
            singularity_message(ERROR, "Failed to resolve path to %s: %s\n", CONTAINER_OVERLAY, strerror(errno));
            ABORT(255);
        }
    }
    if ( container_path->session == NULL ) {
        container_path->session = realpath(SESSIONDIR, NULL); // Flawfinder: ignore
        if ( container_path->session == NULL ) {
            singularity_message(ERROR, "Failed to resolve path to %s: %s\n", SESSIONDIR, strerror(errno));
            ABORT(255);
        }
    }
}

int singularity_mount(const char *source, const char *target,
                      const char *filesystemtype, unsigned long mountflags,
                      const void *data) {
    int ret;
    int mount_errno;
    uid_t fsuid = 0;
    char *realdest;
    static struct resolved_container_path container_path;

    if ( ( mountflags & MS_BIND ) ) {
        fsuid = singularity_priv_getuid();
    }

    realdest = realpath(target, NULL); // Flawfinder: ignore
    if ( realdest == NULL ) {
        singularity_message(ERROR, "Failed to get real path of %s: %s\n", target, strerror(errno));
        ABORT(255);
    }

    resolve_container_path(&container_path);

    if ( (mountflags & MS_PRIVATE) == 0 && (mountflags & MS_SLAVE) == 0 ) {
        if ( strncmp(realdest, container_path.mountdir, strlen(container_path.mountdir)) != 0 &&
             strncmp(realdest, container_path.finaldir, strlen(container_path.finaldir)) != 0 &&
             strncmp(realdest, container_path.overlay, strlen(container_path.overlay)) != 0 &&
             strncmp(realdest, container_path.session, strlen(container_path.session)) != 0 ) {
            singularity_message(VERBOSE, "Ignored, try to mount %s outside of container %s\n", target, realdest);
            free(realdest);
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

    ret = mount(source, realdest, filesystemtype, mountflags, data);
    mount_errno = errno;

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
    char *real_mountpoint = joinpath(CONTAINER_FINALDIR, mountpoint);
    char *resolved_mountpoint = realpath(real_mountpoint, NULL); // Flawfinder: ignore

    if ( resolved_mountpoint == NULL ) {
        free(real_mountpoint);
        return(retval);
    }

    singularity_message(DEBUG, "Checking if currently mounted: %s\n", mountpoint);

    singularity_message(DEBUG, "Opening /proc/mounts\n");
    if ( ( mounts = fopen("/proc/mounts", "r") ) == NULL ) { // Flawfinder: ignore
        singularity_message(ERROR, "Could not open /proc/mounts: %s\n", strerror(errno));
        ABORT(255);
    }

    singularity_message(DEBUG, "Iterating through /proc/mounts\n");
    while ( ( retval < 0 ) && ( fgets(line, MAX_LINE_LEN, mounts) != NULL ) ) {
        (void) strtok(line, " ");
        char *mount = strtok(NULL, " ");

        if ( strcmp(mount, resolved_mountpoint) == 0 ) {
            singularity_message(DEBUG, "Mountpoint is already mounted: %s\n", resolved_mountpoint);
            retval = 1;
        }
    }

    fclose(mounts);
    free(line);
    free(real_mountpoint);
    free(resolved_mountpoint);

    return(retval);
}

