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

#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <errno.h> 
#include <string.h>
#include <fcntl.h>  

#include "config.h"
#include "lib/singularity.h"
#include "util/file.h"
#include "util/util.h"


int main(int argc, char ** argv) {
    long int size;

    if ( argv[1] == NULL ) {
        fprintf(stderr, "USAGE: %s [singularity container image] [increase size in MiB]\n", argv[0]);
        return(1);
    }

    if ( argv[2] == NULL ) {
        size = 1024;
    } else {
        size = ( strtol(argv[2], (char **)NULL, 10) );
    }

    return(singularity_image_expand(argv[1], size));

    return(0);
}
