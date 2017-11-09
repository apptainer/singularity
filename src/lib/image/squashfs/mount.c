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
#include <sys/file.h>
#include <sys/mount.h>
#include <unistd.h>
#include <stdlib.h>

#include "util/file.h"
#include "util/util.h"
#include "util/message.h"
#include "util/config_parser.h"
#include "util/privilege.h"
#include "util/mount.h"

#include "../image.h"
#include "../bind.h"


int _singularity_image_squashfs_mount(struct image_object *image, char *mount_point) {
    char *loop_dev = NULL;

    if ( ( loop_dev = singularity_image_bind(image) ) == NULL ) {
        singularity_message(ERROR, "Could not obtain the image loop device\n");
        ABORT(255);
    }

    singularity_priv_escalate();
    singularity_message(VERBOSE, "Mounting squashfs image: %s -> %s\n", loop_dev, mount_point);
    if ( singularity_mount(loop_dev, mount_point, "squashfs", MS_NOSUID|MS_RDONLY|MS_NODEV, "errors=remount-ro") < 0 ) {
        singularity_message(ERROR, "Failed to mount squashfs image in (read only): %s\n", strerror(errno));
        ABORT(255);
    }
    singularity_priv_drop();

    return(0);
}

