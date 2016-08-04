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


#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/mount.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/file.h>
#include <errno.h> 
#include <string.h>
#include <fcntl.h>
#include <libgen.h>

#include "config.h"
#include "mounts.h"
#include "file.h"
#include "util.h"
#include "loop-control.h"
#include "message.h"

#ifndef MS_REC
#define MS_REC 16384
#endif


int mount_image(char * loop_device, char * mount_point, int writable) {

    message(DEBUG, "Called mount_image(%s, %s, %d)\n", loop_device, mount_point, writable);

    message(DEBUG, "Checking mount point is present\n");
    if ( is_dir(mount_point) < 0 ) {
        message(ERROR, "Mount point is not available: %s\n", mount_point);
        ABORT(255);
    }

    message(DEBUG, "Checking loop is a block device\n");
    if ( is_blk(loop_device) < 0 ) {
        message(ERROR, "Loop device is not a block dev: %s\n", loop_device);
        ABORT(255);
    }

    if ( writable > 0 ) {
        message(DEBUG, "Trying to mount read/write as ext4 with discard option\n");
        if ( mount(loop_device, mount_point, "ext4", MS_NOSUID, "discard,errors=remount-ro") < 0 ) {
            message(DEBUG, "Trying to mount read/write as ext4 without discard option\n");
            if ( mount(loop_device, mount_point, "ext4", MS_NOSUID, "errors=remount-ro") < 0 ) {
                message(DEBUG, "Trying to mount read/write as ext3\n");
                if ( mount(loop_device, mount_point, "ext3", MS_NOSUID, "errors=remount-ro") < 0 ) {
                    message(ERROR, "Failed to mount (rw) '%s' at '%s': %s\n", loop_device, mount_point, strerror(errno));
                    ABORT(255);
                }
            }
        }
    } else {
        message(DEBUG, "Trying to mount read only as ext4 with discard option\n");
        if ( mount(loop_device, mount_point, "ext4", MS_NOSUID|MS_RDONLY, "discard") < 0 ) {
            message(DEBUG, "Trying to mount read only as ext4 without discard option\n");
            if ( mount(loop_device, mount_point, "ext4", MS_NOSUID|MS_RDONLY, "") < 0 ) {
                message(DEBUG, "Trying to mount read only as ext3\n");
                if ( mount(loop_device, mount_point, "ext3", MS_NOSUID|MS_RDONLY, "") < 0 ) {
                    message(ERROR, "Failed to mount (ro) '%s' at '%s': %s\n", loop_device, mount_point, strerror(errno));
                    ABORT(255);
                }
            }
        }
    }

    message(DEBUG, "Returning mount_image(%s, %s, %d) = 0\n", loop_device, mount_point, writable);

    return(0);
}


int mount_bind(char * source, char * dest, int writable) {

    message(DEBUG, "Called mount_bind(%s, %d, %d)\n", source, dest, writable);

    message(DEBUG, "Checking that source exists and is a file or directory\n");
    if ( is_dir(source) != 0 && is_file(source) != 0 ) {
        message(ERROR, "Bind source path is not a file or directory: '%s'\n", source);
        ABORT(255);
    }

    message(DEBUG, "Checking that destination exists and is a file or directory\n");
    if ( is_dir(dest) != 0 && is_file(dest) != 0 ) {
        message(ERROR, "Container bind path is not a file or directory: '%s'\n", dest);
        ABORT(255);
    }

    message(DEBUG, "Calling mount(%s, %s, ...)\n", source, dest);
    if ( mount(source, dest, NULL, MS_BIND|MS_REC, NULL) < 0 ) {
        message(ERROR, "Could not bind %s: %s\n", dest, strerror(errno));
        ABORT(255);
    }

    if ( writable <= 0 ) {
        message(VERBOSE2, "Making mount read only: %s\n", dest);
        if ( mount(NULL, dest, NULL, MS_BIND|MS_REC|MS_REMOUNT|MS_RDONLY, NULL) < 0 ) {
            message(ERROR, "Could not bind read only %s: %s\n", dest, strerror(errno));
            ABORT(255);
        }
    }

    message(DEBUG, "Returning mount_bind(%s, %d, %d) = 0\n", source, dest, writable);

    return(0);
}
