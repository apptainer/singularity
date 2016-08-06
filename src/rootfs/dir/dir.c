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

#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <string.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/mount.h>
#include <unistd.h>
#include <stdlib.h>

#include "file.h"
#include "util.h"
#include "message.h"
#include "dir.h"
#include "privilege.h"


static char *source_dir = NULL;
static char *mount_point = NULL;
static int read_write = 0;


int rootfs_dir_init(char *source, char *mount_dir, int writable) {
    message(DEBUG, "Inializing container rootfs dir subsystem\n");

    if ( is_dir(source) == 0 ) {
        source_dir = strdup(source);
    } else {
        message(ERROR, "Container source directory is not available: %s\n", source);
        ABORT(255);
    }

    if ( is_dir(mount_dir) == 0 ) {
        mount_point = strdup(mount_dir);
    } else {
        message(ERROR, "Mount point for container image is not a directory: %s\n", mount_dir);
        ABORT(255);
    }

    if ( writable > 0 ) {
        read_write =1;
    }

    return(0);
}


int rootfs_dir_mount(void) {

    if ( ( mount_point == NULL ) || ( source_dir == NULL ) ) {
        message(ERROR, "Called image_mount but image_init() hasn't been called\n");
        ABORT(255);
    }

    priv_escalate();
    if ( mount(source_dir, mount_point, NULL, MS_BIND|MS_NOSUID|MS_REC, NULL) < 0 ) {
        message(ERROR, "Could not mount container directory %s->%s: %s\n", source_dir, mount_point, strerror(errno));
        return 1;
    }
    priv_drop();

    return(0);
}

int rootfs_dir_umount(void) {
    if ( mount_point == NULL ) {
        message(ERROR, "Called image_umount but image_init() hasn't been called\n");
        ABORT(255);
    }

    priv_escalate();
    if ( umount(mount_point) < 0 ) {
        message(ERROR, "Failed umounting file system\n");
        ABORT(255);
    }
    priv_drop();

    return(0);
}
