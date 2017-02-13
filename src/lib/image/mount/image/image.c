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
#include "util/registry.h"
#include "util/message.h"
#include "util/config_parser.h"
#include "util/privilege.h"

#include "../../image.h"
#include "../mount.h"


int _singularity_image_mount_image_check(struct image_object *image) {
    return(singularity_image_check(image));
}


int _singularity_image_mount_image_mount(struct image_object *image, char *mount_point) {

    if ( image->loopdev == NULL ) {
        singularity_message(ERROR, "Could not obtain the image loop device for: %s\n", image->path);
        ABORT(255);
    }

    if ( singularity_registry_get("WRITABLE") == NULL ) {
        singularity_priv_escalate();
        singularity_message(VERBOSE, "Mounting %s in read/only to: %s\n", image->loopdev, mount_point);
        if ( mount(image->loopdev, mount_point, "ext3", MS_NOSUID|MS_RDONLY|MS_NODEV, "errors=remount-ro") < 0 ) {
            if ( mount(image->loopdev, mount_point, "ext4", MS_NOSUID|MS_RDONLY|MS_NODEV, "errors=remount-ro") < 0 ) {
                singularity_message(ERROR, "Failed to mount image in (read only): %s\n", strerror(errno));
                ABORT(255);
            }
        }
        singularity_priv_drop();
    } else {
        singularity_priv_escalate();
        singularity_message(VERBOSE, "Mounting %s in read/write to: %s\n", image->loopdev, mount_point);
        if ( mount(image->loopdev, mount_point, "ext3", MS_NOSUID|MS_NODEV, "errors=remount-ro") < 0 ) {
            if ( mount(image->loopdev, mount_point, "ext4", MS_NOSUID|MS_NODEV, "errors=remount-ro") < 0 ) {
                singularity_message(ERROR, "Failed to mount image in (read/write): %s\n", strerror(errno));
                ABORT(255);
            }
        }
        singularity_priv_drop();
    }

    return(0);
}
