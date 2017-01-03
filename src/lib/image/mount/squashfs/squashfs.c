/* 
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
#include <sys/file.h>
#include <sys/mount.h>
#include <unistd.h>
#include <stdlib.h>

#include "util/file.h"
#include "util/util.h"
#include "lib/message.h"
#include "lib/config_parser.h"
#include "lib/privilege.h"

#include "../../image.h"
#include "../mount.h"


int _singularity_image_mount_squashfs_check(void) {
    char *image_name = singularity_image_name();
    int len = strlength(image_name, 1024);

    if ( strcmp(&image_name[len-5], ".sqsh") != 0 ) {
        singularity_message(DEBUG, "Image does not appear to be of type '.sqsh': %s\n", image_name);
        return(-1);
    }

    return(0);
}

int _singularity_image_mount_squashfs_mount(void) {
    char *loop_dev = singularity_image_bind_dev();
    char *mount_point = _singularity_image_mount_sourcepath();

    if ( loop_dev == NULL ) {
        singularity_message(ERROR, "Could not obtain the image loop device\n");
        ABORT(255);
    }

    singularity_priv_escalate();
    singularity_message(VERBOSE, "Mounting squashfs image\n");
    if ( mount(loop_dev, mount_point, "squashfs", MS_NOSUID|MS_RDONLY, "errors=remount-ro") < 0 ) {
        singularity_message(ERROR, "Failed to mount squashfs image in (read only): %s\n", strerror(errno));
        ABORT(255);
    }
    singularity_priv_drop();

    return(0);
}

