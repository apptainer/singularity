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

#include "../../image.h"

int _singularity_image_check_squashfs(struct image_object *image) {
    FILE *image_fp;
    char *line;

    if ( ( image_fp = fdopen(dup(image->fd), "r") ) == NULL ) {
        singularity_message(ERROR, "Could not associate file pointer from file descriptor on image %s: %s\n", image->path, strerror(errno));
        ABORT(255);
    }

    singularity_message(VERBOSE3, "Checking that file pointer is a Singularity image\n");
    rewind(image_fp);

    line = (char *)malloc(5);

    // Get the first line from the config
    if ( fgets(line, 5, image_fp) == NULL ) {
        singularity_message(ERROR, "Unable to read the first 4 bytes of image: %s\n", strerror(errno));
        ABORT(255);
    }

    singularity_message(DEBUG, "found bytes of image: %s\n", line);

    singularity_message(DEBUG, "Checking if first line matches key\n");
    if ( strcmp(line, "hsqs") == 0 ) {
        free(line);
        singularity_message(VERBOSE2, "File is a valid SquashFS image\n");
    } else {
        free(line);
        singularity_message(VERBOSE, "File is not a valid SquashFS image\n");
        return(-1);
    }

    fclose(image_fp);

    return(0);
}
