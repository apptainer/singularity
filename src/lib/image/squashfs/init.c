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

int _singularity_image_squashfs_init(struct image_object *image, int open_flags) {
    int image_fd;
    int ret;
    FILE *image_fp;
    static char buf[1024];
    char *p;

    singularity_message(DEBUG, "Checking if writable image requested\n");
    if ( open_flags == O_RDWR ) {
        errno = EROFS;
        return(-1);
    }
    
    singularity_message(DEBUG, "Opening file descriptor to image: %s\n", image->path);
    if ( ( image_fd = open(image->path, open_flags, 0755) ) < 0 ) {
        singularity_message(ERROR, "Could not open image %s: %s\n", image->path, strerror(errno));
        ABORT(255);
    }

    if ( ( image_fp = fdopen(dup(image_fd), "r") ) == NULL ) {
        singularity_message(ERROR, "Could not associate file pointer from file descriptor on image %s: %s\n", image->path, strerror(errno));
        ABORT(255);
    }

    singularity_message(VERBOSE3, "Checking that file pointer is a Singularity image\n");
    rewind(image_fp);

    // Get the first line from the config
    ret = fread(buf, 1, sizeof(buf), image_fp);
    fclose(image_fp);
    if ( ret != sizeof(buf) ) {
        singularity_message(DEBUG, "Could not read the top of the image\n");
        return(-1);
    }

    singularity_message(DEBUG, "Checking for magic in the top of the file\n");

    /* if LAUNCH_STRING is present, figure out squashfs magic offset */
    p = strstr(buf, "hsqs");
    if ( p != NULL ) {
        singularity_message(VERBOSE2, "File is a valid SquashFS image\n");
        image->offset = p - buf;
    } else {
        close(image_fd);
        singularity_message(VERBOSE, "File is not a valid SquashFS image\n");
        return(-1);
    }

    image->fd = image_fd;

    return(0);
}
