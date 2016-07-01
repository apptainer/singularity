/* 
 * Copyright (c) 2015-2016, Gregory M. Kurtzer. All rights reserved.
 * 
 * “Singularity” Copyright (c) 2016, The Regents of the University of California,
 * through Lawrence Berkeley National Laboratory (subject to receipt of any
 * required approvals from the U.S. Dept. of Energy).  All rights reserved.
 * 
 * If you have questions about your rights to use or distribute this software,
 * please contact Berkeley Lab's Innovation & Partnerships Office at
 * IPO@lbl.gov.
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

#include "file.h"
#include "image.h"
#include "util.h"
#include "message.h"


int image_offset(FILE *image_fp) {
    int ret = 0;
    int i = 0;

    rewind(image_fp);

    for (i=0; i < 64; i++) {
        int c = fgetc(image_fp);
        if ( c == EOF ) {
            break;
        } else if ( c == '\n' ) {
            ret = i + 1;
            break;
        }
    }

    return(ret);
}


int image_create(char *image, int size) {
    FILE *image_fp;
    int i;

    image_fp = fopen(image, "w");
    if ( image_fp == NULL ) {
        fprintf(stderr, "ERROR: Could not open image for writing %s: %s\n", image, strerror(errno));
        return(-1);
    }

    fprintf(image_fp, LAUNCH_STRING);
    for(i = 0; i < size; i++ ) {
        fseek(image_fp, 1024 * 1024, SEEK_CUR);
    }
    fprintf(image_fp, "0");
    fclose(image_fp);

    chmod(image, 0755);

    return(0);
}

int image_expand(char *image, int size) {
    FILE *image_fp;
    long position;

    image_fp = fopen(image, "r+");
    if ( image_fp == NULL ) {
        fprintf(stderr, "ERROR: Could not open image for writing %s: %s\n", image, strerror(errno));
        return(-1);
    }

    fseek(image_fp, 0L, SEEK_END);
    position = ftell(image_fp);
    if ( ftruncate(fileno(image_fp), position-1) < 0 ) {
        fprintf(stderr, "ERROR: Failed truncating the marker bit off of image %s: %s\n", image, strerror(errno));
        return(-1);
    }
    fseek(image_fp, size * 1024 * 1024, SEEK_CUR);
    fprintf(image_fp, "0");
    fclose(image_fp);

    return(0);
}
