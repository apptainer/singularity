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
#include <unistd.h>
#include <stdlib.h>
#include <libgen.h>

#include "util/file.h"
#include "util/util.h"
#include "lib/message.h"

#include "./image.h"
#include "./attach/attach.h"
#include "./bind/bind.h"
#include "./create/create.h"
#include "./check/check.h"
#include "./expand/expand.h"
#include "./offset/offset.h"

static char *temp_directory = NULL;
static char *image_path = NULL;


// extern int singularity_image_expand(char *image, unsigned int size)
//
// extern int singularity_image_mount(char *mountpoint, unsigned int flags);



char *singularity_image_tempdir(char *directory) {
    if ( directory != NULL ) {
        if ( is_dir(directory) == 0 ) {
            temp_directory = strdup(directory);
        } else {
            singularity_message(ERROR, "Temp directory path is not a directory: %s\n", directory);
            ABORT(255);
        }
    }

    return(temp_directory);
}

char *singularity_image_path(char *path) {
    if ( path != NULL ) {
        if ( ( is_file(path) != 0 ) && ( is_dir(path) != 0 ) ) {
            singularity_message(ERROR, "Invalid image path: %s\n", path);
            ABORT(255);
        }
        image_path = strdup(path);
    }
    return(image_path);
}

char *singularity_image_name(void) {
    if ( image_path == NULL ) {
        singularity_message(ERROR, "Image path has not een set\n");
        ABORT(255);
    }
    return(basename(image_path));
}


int singularity_image_attach(void) {
    return(_singularity_image_attach());
}

int singularity_image_attach_fd(void) {
    return(_singularity_image_attach_fd());
}

FILE *singularity_image_attach_fp(void) {
    return(_singularity_image_attach_fp());
}

int singularity_image_create(unsigned int size) {
    return(_singularity_image_create(size));
}

int singularity_image_expand(unsigned int size) {
    return(_singularity_image_expand(size));
}

int singularity_image_check(void) {
    return(_singularity_image_check());
}

int singularity_image_offset(void) {
    return(_singularity_image_offset());
}

int singularity_image_bind(void) {
    return(_singularity_image_bind());
}

char *singularity_image_bind_dev(void) {
    return(_singularity_image_bind_dev());
}
