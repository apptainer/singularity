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

#include "file.h"
#include "image.h"
#include "util.h"
#include "message.h"


int image_offset(FILE *image_fp) {
    int ret = 0;
    int i = 0;

    message(VERBOSE, "Calculating image offset\n");
    rewind(image_fp);

    for (i=0; i < 64; i++) {
        int c = fgetc(image_fp); // Flawfinder: ignore
        if ( c == EOF ) {
            break;
        } else if ( c == '\n' ) {
            ret = i + 1;
            message(VERBOSE2, "Found image at an offset of %d bytes\n", ret);
            break;
        }
    }

    message(DEBUG, "Returning image_offset(image_fp) = %d\n", ret);

    return(ret);
}


int image_create(char *image, int size) {
    FILE *image_fp;
    int i;

    message(VERBOSE, "Creating new sparse image at: %s\n", image);

    if ( is_file(image) == 0 ) {
        message(ERROR, "Will not overwrite existing file: %s\n", image);
        ABORT(255);
    }

    message(DEBUG, "Opening image 'w'\n");
    if ( ( image_fp = fopen(image, "w") ) == NULL ) { // Flawfinder: ignore
        fprintf(stderr, "ERROR: Could not open image for writing %s: %s\n", image, strerror(errno));
        return(-1);
    }

    message(VERBOSE2, "Writing image header\n");
    fprintf(image_fp, LAUNCH_STRING); // Flawfinder: ignore (LAUNCH_STRING is a constant)

    message(VERBOSE2, "Expanding image to %dMB\n", size);
    for(i = 0; i < size; i++ ) {
        fseek(image_fp, 1024 * 1024, SEEK_CUR);
    }
    fprintf(image_fp, "0");

    message(VERBOSE2, "Making image executable\n");
    fchmod(fileno(image_fp), 0755);

    fclose(image_fp);

    message(DEBUG, "Returning image_create(%s, %d) = 0\n", image, size);

    return(0);
}

int image_expand(char *image, int size) {
    FILE *image_fp;
    long position;
    int i;

    message(VERBOSE, "Expanding sparse image at: %s\n", image);

    message(DEBUG, "Opening image 'r+'\n");
    if ( ( image_fp = fopen(image, "r+") ) == NULL ) { // Flawfinder: ignore
        fprintf(stderr, "ERROR: Could not open image for writing %s: %s\n", image, strerror(errno));
        return(-1);
    }

    message(DEBUG, "Jumping to the end of the current image file\n");
    fseek(image_fp, 0L, SEEK_END);
    position = ftell(image_fp);

    message(DEBUG, "Removing the footer from image\n");
    if ( ftruncate(fileno(image_fp), position-1) < 0 ) {
        fprintf(stderr, "ERROR: Failed truncating the marker bit off of image %s: %s\n", image, strerror(errno));
        return(-1);
    }
    message(VERBOSE2, "Expanding image by %dMB\n", size);
    for(i = 0; i < size; i++ ) {
        fseek(image_fp, 1024 * 1024, SEEK_CUR);
    }
    fprintf(image_fp, "0");
    fclose(image_fp);

    message(DEBUG, "Returning image_expand(%s, %d) = 0\n", image, size);

    return(0);
}
