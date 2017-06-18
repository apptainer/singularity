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

#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <errno.h> 
#include <string.h>
#include <fcntl.h>  

#include "util/message.h"
#include "util/util.h"
#include "util/file.h"

#include "../image.h"

#define BUFFER_SIZE (1024*1024)

int _singularity_image_create(struct image_object *image, long int size) {
    FILE *image_fp;
    int retval;

    if ( image->fd <= 0 ) {
        singularity_message(ERROR, "Can not check image with no FD associated\n");
        ABORT(255);
    }

    if ( ( image_fp = fdopen(dup(image->fd), "w") ) == NULL ) {
        singularity_message(ERROR, "Could not associate file pointer from file descriptor on image %s: %s\n", image->path, strerror(errno));
        ABORT(255);
    }

    singularity_message(VERBOSE2, "Writing image header\n");
    fprintf(image_fp, LAUNCH_STRING); // Flawfinder: ignore (LAUNCH_STRING is a constant)

    singularity_message(VERBOSE2, "Growing image to %ldMB\n", size);
    while ( 1 ) {
        retval = posix_fallocate(singularity_image_fd(image), sizeof(LAUNCH_STRING), size * BUFFER_SIZE);

        if ( retval == EINTR ) {
            singularity_message(DEBUG, "fallocate was interrupted by a signal, trying again...\n");
            continue;
        } else {
            break;
        }
    }

    if ( retval != 0 ) {
        switch ( retval ) {
            case ENOSPC:
                singularity_message(ERROR, "There is not enough to space to allocate the image\n");
                break;
            case EBADF:
                singularity_message(ERROR, "The image file descriptor is not valid for writing\n");
                break;
            case EFBIG:
                singularity_message(ERROR, "The image size was too big for the filesystem\n");
                break;
            case EINVAL:
                singularity_message(ERROR, "The image size is invalid.\n");
                break;
        }
        ABORT(255);
    }

    fclose(image_fp);

    return(0);
}
