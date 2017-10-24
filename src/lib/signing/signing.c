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

#include "crypt.h"
#include "util/message.h"
#include "util/file.h"
#include "util/util.h"

#include "../sif/list.h"
#include "../sif/sif.h"

extern char verifblock[VERIFBLOCK_SIZE];

int _singularity_image_sign(unsigned char *text) {
#if 0
    ssize_t retval;
    unsigned char *map;
    static unsigned char hash[HASH_LEN];
    static char hashstr[sizeof(IMAGE_HASH_PREFIX)+HASH_LEN*2+1];

    if ( image->fd <= 0 ) {
        singularity_message(ERROR, "Can not check image with no FD associated\n");
        ABORT(255);
    }

    if ( ( image_fp = fdopen(dup(image->fd), "w") ) == NULL ) {
        singularity_message(ERROR, "Could not associate file pointer from file descriptor on image %s: %s\n",
                            image->path, strerror(errno));
        ABORT(255);
    }

    singularity_message(DEBUG, "Computing hash from '%c' for %ld bytes\n", map[0], image->size);
    compute_hash(map, image->size, hash);
    strcpy(hashstr, IMAGE_HASH_PREFIX);
    for (int i = 0, pos = strlen(IMAGE_HASH_PREFIX); i < HASH_LEN; i++, pos = i*2+strlen(IMAGE_HASH_PREFIX)) {
        sprintf(&hashstr[pos], "%02hhx", hash[i]);
    }
    sign_verifblock(hashstr, verifblock);
    singularity_message(DEBUG, "Writing verification block to image's end\n");
#endif
    return(0);
}

int _singularity_image_verify(unsigned char *signature) {
#if 0
    int ret;
    char *vb, *vb_hashstr;
    unsigned char *map;
    int pgoff = image->vboff - (image->vboff & ~(0xFFF));
    int is_equal = 1;
    static unsigned char hash[HASH_LEN];
    static char hashstr[sizeof(IMAGE_HASH_PREFIX)+HASH_LEN*2+1];

    vb = mmap_file(image->vboff - pgoff, sysconf(_SC_PAGESIZE)*2, image->fd);
    if (strncmp(&vb[pgoff], VERIFBLOCK_MAGIC, strlen(VERIFBLOCK_MAGIC))) {
        singularity_message(ERROR, "Could not find PGP signature at verification block\n");
        ABORT(255);
    }

    ret = verify_verifblock(&vb[pgoff]);
    if (ret < 0) {
        singularity_message(ERROR, "Signature is not good\n");
        munmap_file(vb, sysconf(_SC_PAGESIZE)*2);
        return -1;
    } else {
        singularity_message(INFO, "Signature is good\n");
    }

    vb_hashstr = strstr(&vb[pgoff], IMAGE_HASH_PREFIX);
    if (vb_hashstr == NULL) {
        singularity_message(ERROR, "Could not locate image hash\n");
        ABORT(255);
    }
    vb_hashstr += strlen(IMAGE_HASH_PREFIX);

    map = mmap_file(0, image->size, image->fd);
    singularity_message(DEBUG, "Computing hash from '%c' for %ld bytes\n", map[0], image->size);
    compute_hash(map, image->size, hash);

    for (int i = 0; i < HASH_LEN; i++) {
        sprintf(&hashstr[i*2], "%02hhx", hash[i]);
    }

    if (strncmp(hashstr, vb_hashstr, HASH_LEN*2)) {
        is_equal = 0;
        singularity_message(ERROR, "Image hashes don't match\n");
    } else {
        singularity_message(INFO, "Image hashes match\n");
    }

    munmap_file(map, image->size);
    munmap_file(vb, sysconf(_SC_PAGESIZE)*2);

    if (!is_equal)
        return -1;
#endif

    return 0;
}
