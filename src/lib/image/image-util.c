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
#include "lib/image-util.h"


#define LAUNCH_STRING "#!/usr/bin/env run-singularity\n"
#define MAX_LINE_LEN 2048


int singularity_image_check(FILE *image_fp) {
    char *line;

    if ( image_fp == NULL ) {
        singularity_message(ERROR, "Called singularity_image_check() with NULL image pointer\n");
        ABORT(255);
    }

    singularity_message(VERBOSE3, "Checking file is a Singularity image\n");
    rewind(image_fp);

    line = (char *)malloc(MAX_LINE_LEN);

    // Get the first line from the config
    if ( fgets(line, MAX_LINE_LEN, image_fp) == NULL ) {
        singularity_message(ERROR, "Unable to read the first line of image: %s\n", strerror(errno));
        ABORT(255);
    }

    singularity_message(DEBUG, "Checking if first line matches key\n");
    if ( strcmp(line, LAUNCH_STRING) == 0 ) {
        free(line);
        singularity_message(VERBOSE2, "File is a valid Singularity image\n");
    } else {
        free(line);
        singularity_message(VERBOSE, "File is not a valid Singularity image\n");
        return(-1);
    }

    return(0);
}


int singularity_image_offset(FILE *image_fp) {
    int ret = 0;
    int i = 0;

    singularity_message(VERBOSE, "Calculating image offset\n");
    rewind(image_fp);

    for (i=0; i < 64; i++) {
        int c = fgetc(image_fp); // Flawfinder: ignore
        if ( c == EOF ) {
            break;
        } else if ( c == '\n' ) {
            ret = i + 1;
            singularity_message(VERBOSE2, "Found image at an offset of %d bytes\n", ret);
            break;
        }
    }

    singularity_message(DEBUG, "Returning image_offset(image_fp) = %d\n", ret);

    return(ret);
}


int singularity_image_create(char *image, int size) {
    FILE *image_fp;
    char *buff = (char *) malloc(1024*1024);
    int i;

    singularity_message(VERBOSE, "Creating new sparse image at: %s\n", image);

    if ( is_file(image) == 0 ) {
        singularity_message(ERROR, "Will not overwrite existing file: %s\n", image);
        ABORT(255);
    }

    singularity_message(DEBUG, "Opening image 'w'\n");
    if ( ( image_fp = fopen(image, "w") ) == NULL ) { // Flawfinder: ignore
        fprintf(stderr, "ERROR: Could not open image for writing %s: %s\n", image, strerror(errno));
        return(-1);
    }

    singularity_message(VERBOSE2, "Writing image header\n");
    fprintf(image_fp, LAUNCH_STRING); // Flawfinder: ignore (LAUNCH_STRING is a constant)

    singularity_message(VERBOSE2, "Expanding image to %dMB\n", size);
    for(i = 0; i < size; i++ ) {
        if ( fwrite(buff, 1, 1024*1024, image_fp) < 1024 * 1024 ) {
            singularity_message(ERROR, "Failed allocating space to image: %s\n", strerror(errno));
            ABORT(255);
        }
    }

    singularity_message(VERBOSE2, "Making image executable\n");
    fchmod(fileno(image_fp), 0755);

    fclose(image_fp);

    singularity_message(DEBUG, "Returning image_create(%s, %d) = 0\n", image, size);

    return(0);
}

int singularity_image_expand(char *image, int size) {
    FILE *image_fp;
    char *buff = (char *) malloc(1024*1024);
    long position;
    int i;

    singularity_message(VERBOSE, "Expanding sparse image at: %s\n", image);

    singularity_message(DEBUG, "Opening image 'r+'\n");
    if ( ( image_fp = fopen(image, "r+") ) == NULL ) { // Flawfinder: ignore
        fprintf(stderr, "ERROR: Could not open image for writing %s: %s\n", image, strerror(errno));
        return(-1);
    }

    singularity_message(DEBUG, "Jumping to the end of the current image file\n");
    fseek(image_fp, 0L, SEEK_END);
    position = ftell(image_fp);

    singularity_message(DEBUG, "Removing the footer from image\n");
    if ( ftruncate(fileno(image_fp), position-1) < 0 ) {
        fprintf(stderr, "ERROR: Failed truncating the marker bit off of image %s: %s\n", image, strerror(errno));
        return(-1);
    }
    singularity_message(VERBOSE2, "Expanding image by %dMB\n", size);
    for(i = 0; i < size; i++ ) {
        if ( fwrite(buff, 1, 1024*1024, image_fp) < 1024 * 1024 ) {
            singularity_message(ERROR, "Failed allocating space to image: %s\n", strerror(errno));
            ABORT(255);
        }
    }
    fprintf(image_fp, "0");
    fclose(image_fp);

    singularity_message(DEBUG, "Returning image_expand(%s, %d) = 0\n", image, size);

    return(0);
}
