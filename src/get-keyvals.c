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
    FILE *bootdef_fp;
    char *line;
    char *path;

    if ( argc < 2 ) {
        printf("USAGE: %s [file]\n", argv[0]);
        exit(0);
    }

    path = argv[1];

    if ( is_file(path) != 0 ) {
        singularity_message(ERROR, "Bootstrap definition file not found: %s\n", path);
        ABORT(255);
    }

    if ( ( bootdef_fp = fopen(path, "r") ) == NULL ) {
        singularity_message(ERROR, "Could not open bootstrap definition file %s: %s\n", path, strerror(errno));
        ABORT(255);
    }

    line = (char *)malloc(MAX_LINE_LEN);

    while ( fgets(line, MAX_LINE_LEN, bootdef_fp) ) {
        char *bootdef_key;

        if ( line[0] == '%' ) { // We hit a section, stop parsing for keyword tags
            break;
        } else if ( ( bootdef_key = strtok(line, ":") ) != NULL ) {
            char *bootdef_value;

            chomp(bootdef_key);

            if ( ( bootdef_value = strtok(NULL, "\n") ) != NULL ) {

                chomp_comments(bootdef_value);
                singularity_message(VERBOSE2, "Got bootstrap definition key/val '%s' = '%s'\n", bootdef_key, bootdef_value);

                printf("declare -x '%s'='%s'\n", bootdef_key, bootdef_value);
            }
        }
    }

    free(line);
    fclose(bootdef_fp);

    return(0);
}
