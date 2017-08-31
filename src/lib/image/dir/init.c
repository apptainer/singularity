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

int _singularity_image_dir_init(struct image_object *image, int open_flags) {
    int fd = -1;
    struct stat st;

    singularity_message(DEBUG, "Opening file descriptor to directory: %s\n", image->path);
    if ( ( fd = open(image->path, O_RDONLY, 0755) ) < 0 ) {
        singularity_message(ERROR, "Could not open image %s: %s\n", image->path, strerror(errno));
        ABORT(255);
    }

    if ( fstat(fd, &st) != 0 ) {
        singularity_message(ERROR, "Could not stat file descriptor: %s\n", strerror(errno));
        ABORT(255);
    }

    if ( S_ISDIR(st.st_mode) == 0 ) {
        singularity_message(DEBUG, "This is not a directory based image\n");
        close(fd);
        return(-1);
    }

    // If we got here, we assume things are a directory
    image->fd = fd;

    return(0);
}
