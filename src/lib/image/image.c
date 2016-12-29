/* 
 * Copyright (c) 2015-2016, Gregory M. Kurtzer. All rights reserved.
 * 
 * “Singularity” Copyright (c) 2016, The Regents of the University of California,
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

#include "util/file.h"
#include "util/util.h"
#include "lib/message.h"

#include "./image.h"
#include "./attach/attach.h"
#include "./create/create.h"
#include "./check/check.h"
#include "./expand/expand.h"

#include "./offset/offset.h"

//#define MAX_LINE_LEN 2048

// extern int singularity_image_expand(char *image, unsigned int size)
//
// extern int singularity_image_mount(char *mountpoint, unsigned int flags);
//

int singularity_image_attach(char *image) {
    return(_singularity_image_attach(image));
}

int singularity_image_attach_fd(void) {
    return(_singularity_image_attach_fd());
}

FILE *singularity_image_attach_fp(void) {
    return(_singularity_image_attach_fp());
}

int singularity_image_create(char *image, unsigned int size) {
    return(_singularity_image_create(image, size));
}

int singularity_image_expand(FILE *image_fp, unsigned int size) {
    return(_singularity_image_expand(image_fp, size));
}

int singularity_image_check(FILE *image_fp) {
    return(_singularity_image_check(image_fp));
}

int singularity_image_offset(FILE *image_fp) {
    return(_singularity_image_offset(image_fp));
}

int singularity_image_bind(FILE *image_fp) {
    return(_singularity_image_bind(image_fp));
}

char *singularity_image_bind_dev(void) {
    return(_singularity_image_bind_dev());
}
