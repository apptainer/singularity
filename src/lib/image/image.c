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
#include <unistd.h>
#include <stdlib.h>
#include <libgen.h>

#include "util/file.h"
#include "util/util.h"
#include "util/message.h"
#include "util/registry.h"

#include "./image.h"
#include "./bind.h"
#include "./squashfs/include.h"
#include "./dir/include.h"
#include "./ext3/include.h"


struct image_object singularity_image_init(char *path, int open_flags) {
    struct image_object image;

    if ( path == NULL ) {
        singularity_message(ERROR, "No container image path defined\n");
        ABORT(255);
    }

    image.path = strdup(path);
    image.name = basename(strdup(path));
    image.type = -1;
    image.fd = -1;
    image.loopdev = NULL;
    image.offset = 0;

    if ( open_flags & ( O_RDWR | O_WRONLY ) ) {
        image.writable = 1;
    } else {
        image.writable = 0;
    }

    singularity_message(DEBUG, "Calling image_init for each file system module\n");
    if ( _singularity_image_dir_init(&image, open_flags) == 0 ) {
        singularity_message(DEBUG, "got image_init type for directory\n");
        image.type = DIRECTORY;
    } else if ( _singularity_image_squashfs_init(&image, open_flags) == 0 ) {
        singularity_message(DEBUG, "got image_init type for squashfs\n");
        image.type = SQUASHFS;
    } else if ( _singularity_image_ext3_init(&image, open_flags) == 0 ) {
        singularity_message(DEBUG, "got image_init type for ext3\n");
        image.type = EXT3;
    } else {
        singularity_message(ERROR, "Unknown image format/type: %s\n", path);
        ABORT(255);
    }

    return(image);
}

int singularity_image_fd(struct image_object *image) {
    return(image->fd);
}

char *singularity_image_loopdev(struct image_object *image) {
    return(image->loopdev);
}

char *singularity_image_name(struct image_object *image) {
    return(image->name);
}

char *singularity_image_path(struct image_object *image) {
    return(image->path);
}

int singularity_image_offset(struct image_object *image) {
    return(image->offset);
}

int singularity_image_type(struct image_object *image) {
    return(image->type);
}

int singularity_image_writable(struct image_object *image) {
    return(image->writable);
}

int singularity_image_mount(struct image_object *image, char *mount_point) {
    if ( singularity_registry_get("DAEMON_JOIN") ) {
        singularity_message(ERROR, "Internal Error - This function should not be called when joining an instance\n");
    }

    singularity_message(DEBUG, "Figuring out which mount module to use...\n");
    if ( image->type == SQUASHFS ) {
        singularity_message(DEBUG, "Calling squashfs_mount\n");
        return(_singularity_image_squashfs_mount(image, mount_point));
    } else if ( image->type == DIRECTORY ) {
        singularity_message(DEBUG, "Calling dir_mount\n");
        return(_singularity_image_dir_mount(image, mount_point));
    } else if ( image->type == EXT3 ) {
        singularity_message(DEBUG, "Calling ext3_mount\n");
        return(_singularity_image_ext3_mount(image, mount_point));
    } else {
        singularity_message(ERROR, "Can not mount file system of unknown type\n");
        ABORT(255);
    }
    return(-1);
}
