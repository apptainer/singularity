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


#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <errno.h>
#include <string.h>
#include <fcntl.h>

#include "config.h"
#include "util/util.h"
#include "util/config_parser.h"
#include "util/message.h"
#include "lib/image/image.h"

#define MAX_LINE_LEN 4096


int main(int argc, char **argv) {
    struct image_object image;

    singularity_config_init();

    singularity_message(VERBOSE3, "Instantiating read only container image object\n");
    image = singularity_image_init(argv[1], O_RDONLY);


    if ( singularity_image_type(&image) == SQUASHFS ) {
        printf("SQUASHFS\n");
    } else if ( singularity_image_type(&image) == EXT3 ) {
        printf("EXT3\n");
    } else if ( singularity_image_type(&image) == DIRECTORY ) {
        printf("DIRECTORY\n");
    } else {
        singularity_message(ERROR, "Unknown image type\n");
        return(1);
    }

    return(0);
}
