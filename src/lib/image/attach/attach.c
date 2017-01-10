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
#include <limits.h>
#include <unistd.h>
#include <stdlib.h>
#include <grp.h>
#include <pwd.h>
#include <libgen.h>

#include "util/file.h"
#include "util/util.h"
#include "lib/config_parser.h"
#include "lib/message.h"
#include "lib/privilege.h"

#include "../image.h"

static FILE *image_fp;


int _singularity_image_attach(void) {
    char *image = singularity_image_path(NULL);

    singularity_message(DEBUG, "Checking if image is set\n");
    if ( image_fp != NULL ) {
        singularity_message(ERROR, "Call to singularity_image_attach() when already attached!\n");
        ABORT(255);
    }

    singularity_message(DEBUG, "Checking if image is a file: %s\n", image);
    if ( is_file(image) == 0 ) {
        singularity_message(DEBUG, "Obtaining file pointer to image\n");
        image_fp = fopen(image, "r");

        if ( image_fp == NULL ) {
            singularity_message(ERROR, "Could not open image %s: %s\n", image, strerror(errno));
            ABORT(255);
        }
    }

    return(fileno(image_fp));
}

int _singularity_image_attach_fd(void) {
    if ( image_fp == NULL ) {
        singularity_message(ERROR, "Singularity image FD requested, but not attached!\n");
        ABORT(255);
    }
    return(fileno(image_fp));
}


FILE *_singularity_image_attach_fp(void) {
    if ( image_fp == NULL ) {
        singularity_message(ERROR, "Singularity image FD requested, but not attached!\n");
        ABORT(255);
    }
    return(image_fp);
}
