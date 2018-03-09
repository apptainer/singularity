/*
 * Copyright (c) 2017-2018, SyLabs, Inc. All rights reserved.
 * Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
 *
 * This software is licensed under a 3-clause BSD license.  Please
 * consult LICENSE file distributed with the sources of this project regarding
 * your rights to use or distribute this software.
 *
 */

#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <errno.h>
#include <string.h>
#include <fcntl.h>

#include <uuid/uuid.h>

#include "util/message.h"
#include "util/util.h"
#include "util/file.h"

#include "sif/list.h"
#include "sif/sif.h"
#include "sif/sifaccess.h"
#include "image/image.h"

int _singularity_image_sif_init(struct image_object *image, int open_flags) {
    Sifinfo sif;
    Sifpartition *partdesc;
    int rdonly = 1;

    singularity_message(DEBUG, "Checking if writable image requested\n");
    if ( open_flags & O_RDWR ) {
        rdonly = 0;
    }

    if (sif_load(image->path, &sif, rdonly) < 0) {
        singularity_message(VERBOSE, "File is not a valid SIF image\n");
        return(-1);
    } else {
        singularity_message(VERBOSE2, "File is a valid SIF image\n");
    }

    if ( singularity_message_level() >= VERBOSE3 )
        sif_printheader(&sif);

    partdesc = sif_getpartition(&sif, SIF_DEFAULT_GROUP);
    if ( partdesc == NULL ) {
        singularity_message(ERROR, "%s\n", sif_strerror(siferrno));
        return(-1);
    }

    image->offset = partdesc->cm.fileoff;
    image->size = partdesc->cm.filelen;
    image->fd = sif.fd;
    switch(partdesc->fstype){
    case FS_SQUASH:
        image->type = SQUASHFS;
        break;
    case FS_EXT3:
        image->type = EXT3;
        break;
    default:
        singularity_message(ERROR, "Don't know how to handle that partition type\n");
        return(-1);
    }

    return(0);
}
