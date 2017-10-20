/*
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

#include "util/crypt.h"
#include "util/message.h"
#include "util/file.h"
#include "util/util.h"

#include "../sif/list.h"
#include "../sif/sif.h"
#include "../image.h"

extern char verifblock[VERIFBLOCK_SIZE];

int _singularity_image_sign(struct image_object *image) {
    FILE *image_fp;
#if 0
    ssize_t retval;
    unsigned char *map;
    static unsigned char hash[HASH_LEN];
    static char hashstr[sizeof(IMAGE_HASH_PREFIX)+HASH_LEN*2+1];
#endif

    if ( image->fd <= 0 ) {
        singularity_message(ERROR, "Can not check image with no FD associated\n");
        ABORT(255);
    }

    if ( ( image_fp = fdopen(dup(image->fd), "w") ) == NULL ) {
        singularity_message(ERROR, "Could not associate file pointer from file descriptor on image %s: %s\n",
                            image->path, strerror(errno));
        ABORT(255);
    }

#if 0
    singularity_message(DEBUG, "Computing hash from '%c' for %ld bytes\n", map[0], image->size);
    compute_hash(map, image->size, hash);
    strcpy(hashstr, IMAGE_HASH_PREFIX);
    for (int i = 0, pos = strlen(IMAGE_HASH_PREFIX); i < HASH_LEN; i++, pos = i*2+strlen(IMAGE_HASH_PREFIX)) {
        sprintf(&hashstr[pos], "%02hhx", hash[i]);
    }
    sign_verifblock(hashstr, verifblock);
    singularity_message(DEBUG, "Writing verification block to image's end\n");
#endif

    fclose(image_fp);

    return(0);
}
