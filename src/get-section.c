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
#include <sys/stat.h>
#include <sys/types.h>
#include <errno.h> 
#include <string.h>
#include <fcntl.h>  
#include <linux/limits.h>

#include "config.h"
#include "util/file.h"
#include "util/util.h"


#define MAX_LINE_LEN 4096


int main(int argc, char ** argv) {
    char *section;
    char *file;
    int toggle_section = 0;
    int retval = 1;
    FILE *input;
    char *line = (char *)malloc(MAX_LINE_LEN);

    if ( argc < 3 ) {
        printf("USAGE: %s [section] [file]\n", argv[0]);
        exit(0);
    }

    section = strdup(argv[1]);
    file = strdup(argv[2]);

    if ( is_file(file) < 0 ) {
        singularity_message(ERROR, "File not found: %s\n", file);
        ABORT(1);
    }

    if ( ( input = fopen(file, "r") ) == NULL ) { // Flawfinder: ignore
        singularity_message(ERROR, "Could not open file %s: %s\n", file, strerror(errno));
        ABORT(255);
    }

    singularity_message(DEBUG, "Iterating through file looking for sections matching: %%%s\n", section);
    while ( fgets(line, MAX_LINE_LEN, input) != NULL ) {
        if ( strncmp(line, strjoin("%", section), strlength(section, 128) + 1) == 0 ) {
            toggle_section = 1;
            retval = 0;
        } else if ( ( toggle_section == 1 ) && ( strncmp(line, "%", 1) == 0 ) ) {
            toggle_section = 0;
        } else if ( toggle_section == 1 ) {
            printf("%s", line);
        }
    }
    fclose(input);
    free(section);
    free(file);
    free(line);
    return(retval);
}
