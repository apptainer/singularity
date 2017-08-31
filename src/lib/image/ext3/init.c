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
#include <linux/limits.h>

#include "util/message.h"
#include "util/util.h"
#include "util/file.h"
#include "util/registry.h"

#include "../image.h"

#define BUFFER_SIZE     (1024*1024)
#define MAX_LINE_LEN    2048


int _singularity_image_ext3_init(struct image_object *image, int open_flags) {
    int image_fd;
    char *line;
    FILE *image_fp;
    char *image_name = image->name;
    int image_name_len = strlength(image_name, PATH_MAX);

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

    line = (char *)malloc(MAX_LINE_LEN);

    // Get the first line from the config
    if ( fgets(line, MAX_LINE_LEN, image_fp) == NULL ) {
        singularity_message(DEBUG, "Unable to read the first line of image\n");
        return(-1);
    }

    fclose(image_fp);

    singularity_message(DEBUG, "First line of image(fd=%d): %s\n", image->fd, line);

    singularity_message(DEBUG, "Checking if first line matches key\n");
    if ( strcmp(line, LAUNCH_STRING) == 0 ) {
        // Get the image offset
        image->offset = strlength(line, MAX_LINE_LEN);
        singularity_message(DEBUG, "Got image offset: %d\n", image->offset);

        free(line);
        singularity_message(VERBOSE2, "File is a valid Singularity image\n");
    } else {
        free(line);
        if ( strncmp(&image_name[image_name_len - 4], ".img", 4) == 0 ) {
            singularity_message(VERBOSE, "Image has no header, trusting suffix\n");
        } else {
            close(image_fd);
            singularity_message(VERBOSE, "File is not a valid Singularity image\n");
            return(-1);
        }
    }

    image->fd = image_fd;

    return(0);
}
