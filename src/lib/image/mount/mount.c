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
#include <sys/types.h>
#include <sys/mount.h>
#include <unistd.h>
#include <stdlib.h>
#include <libgen.h>
#include <linux/limits.h>

#include "util/file.h"
#include "util/util.h"
#include "util/registry.h"
#include "util/message.h"
#include "util/config_parser.h"
#include "util/privilege.h"

#include "../image.h"
#include "./image/image.h"
#include "./dir/dir.h"
#include "./squashfs/squashfs.h"



int _singularity_image_mount(struct image_object *image, char *mount_point) {

    if ( mount_point == NULL ) {
        singularity_message(ERROR, "Mount point location must exist\n");
        ABORT(255);
    }

    if ( chk_mode(mount_point, 0040755, 0007000) != 0 ) {
        int ret;
        singularity_message(DEBUG, "fixing bad permissions on %s\n", mount_point);

        singularity_priv_escalate();
        ret = chmod(mount_point, 0755); // Flawfinder: ignore (TOCTOU preferred to opening attack surface with priviledged file descriptor)
        singularity_priv_drop();

        if ( ret != 0 ) {
            singularity_message(ERROR, "Bad permission mode (should be 0755) on: %s\n", mount_point);
            ABORT(255);
        }
    }

    singularity_message(VERBOSE, "Checking what kind of image we are mounting\n");
    if ( _singularity_image_mount_squashfs_check(image) == 0 ) {
        if ( _singularity_image_mount_squashfs_mount(image, mount_point) < 0 ) {
            singularity_message(ERROR, "Failed mounting image, aborting...\n");
            ABORT(255);
        }
    } else if ( _singularity_image_mount_dir_check(image) == 0 ) {
        if ( _singularity_image_mount_dir_mount(image, mount_point) < 0 ) {
            singularity_message(ERROR, "Failed mounting image, aborting...\n");
            ABORT(255);
        }
    } else {
        singularity_message(VERBOSE, "Attempting to mount as singularity image\n");
        if ( _singularity_image_mount_image_mount(image, mount_point) < 0 ) {
            singularity_message(ERROR, "Failed mounting image, aborting...\n");
            ABORT(255);
        }
    }

    return(0);
}
