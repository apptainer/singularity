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
#include "./image/image.h"
#include "./dir/dir.h"
#include "./squashfs/squashfs.h"

#define BUFFER_SIZE     (1024*1024)
#define MAX_LINE_LEN    2048


int _singularity_image_check(struct image_object *image) {

    singularity_message(VERBOSE, "Checking what kind of image we are mounting\n");

    if ( _singularity_image_check_image(image) == 0 ) {
        image->type = SINGULARITY;
    } else if ( _singularity_image_check_squashfs(image) == 0 ) {
        image->type = SQUASHFS;
    } else if ( _singularity_image_check_dir(image) == 0 ) {
        image->type = DIRECTORY;
    } else {
        singularity_message(ERROR, "Unknown image format/type.\n");
        ABORT(255);
    }

    return(0);
}

