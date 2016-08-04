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
#include <unistd.h>
#include <stdlib.h>

#include "file.h"
#include "image.h"
#include "util.h"
#include "message.h"
#include "rootfs/rootfs.h"
#include "rootfs/image/image.h"
#include "rootfs/dir/dir.h"

int module = 0;

int rootfs_init(char *source, char *mount_point, int writable) {

    if ( is_file(source) == 0 ) {
        module = ROOTFS_IMAGE;
        return(rootfs_image_init(source, mount_point, writable));
    } else if ( is_dir(source) == 0 ) {
        module = ROOTFS_DIR;
        return(rootfs_dir_init(source, mount_point, writable));
    }

    message(ERROR, "Unknown rootfs source type\n");
    return(-1);
}

int rootfs_mount(void) {

    if ( module == ROOTFS_IMAGE ) {
        return(rootfs_image_mount());
    } else if ( module == ROOTFS_DIR ) {
        return(rootfs_dir_mount());
    }

    message(ERROR, "Called rootfs_mount() without rootfs_init()\n");
    return(-1);
}

int rootfs_umount(void) {

    if ( module == ROOTFS_IMAGE ) {
        return(rootfs_image_umount());
    } else if ( module == ROOTFS_DIR ) {
        return(rootfs_dir_umount());
    }

    message(ERROR, "Called rootfs_umount() without rootfs_init()\n");
    return(-1);
}
