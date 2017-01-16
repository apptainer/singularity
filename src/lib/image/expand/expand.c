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

#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <errno.h> 
#include <string.h>
#include <fcntl.h>  

#include "util/message.h"
#include "util/file.h"
#include "util/util.h"

#include "../image.h"

#define BUFFER_SIZE (1024*1024)


int _singularity_image_expand(struct image_object *image, unsigned int size) {
    int i;
    char *buff = (char *) malloc(BUFFER_SIZE);
    FILE *image_fp;

    if ( singularity_image_check(image) != 0 ) {
        singularity_message(ERROR, "File does not seem to be a valid Singularity image: %s\n", image->path);
        ABORT(255);
    }

    if ( ( image_fp = fdopen(image->fd, "r+") ) == NULL ) {
        singularity_message(ERROR, "Could not fdopen() image file descriptor for %s: %s\n", image->path, strerror(errno));
        ABORT(255);
    }

    memset(buff, '\255', BUFFER_SIZE);

    if ( image_fp == NULL ) {
        singularity_message(ERROR, "Called _singularity_image_expand() with NULL image pointer\n");
        ABORT(255);
    }

    singularity_message(DEBUG, "Jumping to the end of the current image file\n");
    fseek(image_fp, 0L, SEEK_END);

    singularity_message(VERBOSE2, "Expanding image by %dMB\n", size);
    for(i = 0; i < size; i++ ) {
        if ( fwrite(buff, 1, BUFFER_SIZE, image_fp) < BUFFER_SIZE ) {
            singularity_message(ERROR, "Failed allocating space to image: %s\n", strerror(errno));
            ABORT(255);
        }
    }

    fclose(image_fp);
    free(buff);

    return(0);
}

