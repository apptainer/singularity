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

#include "file.h"


int main(int argc, char **argv) {
    FILE *fd;
    char data[1024]; // Flawfinder: ignore
    int i;

    if ( argv[1] == NULL ) {
        fprintf(stderr, "USAGE: %s /path/to/file/to/check\n", argv[0]);
        return(255);
    }


    if ( is_file(argv[1] ) != 0 && is_link(argv[1] ) != 0 ) {
        printf("is not file: %s\n", argv[1]);
        return(255);
    }

    if ( ( fd = fopen(argv[1], "r") ) == NULL ) { // Flawfinder: ignore
        fprintf(stderr, "ERROR: Could not open file %s: %s\n", argv[1], strerror(errno));
        return(255);
    }

    if ( is_exec(argv[1]) == 0 ) {
        for(i=0; i<128; i++) {
            data[i] = fgetc(fd); // Flawfinder: ignore
        }

        if ( strncmp(data, "#!/", 3) == 0 ) {
            char sub[128]; // Flawfinder: ignore
            int a;

            for(a=0;a<128;a++) {
                if ( data[a+2] == '\n' ) {
                    sub[a] = '\0';
                    break;
                }
                sub[a] = data[a+2];
            }

            printf("exe-ascii \"%s\"\n", sub);
        } else {
            for(i=128; i<1024; i++) {
                data[i] = fgetc(fd); // Flawfinder: ignore
            }

            if ( memchr(data, '\0', 1024) != NULL ) {
                printf("exe-binary data\n");
            } else {
                printf("exe-ascii data\n");
            }
        }
    } else {
        for(i=0; i<1024; i++) {
            data[i] = fgetc(fd); // Flawfinder: ignore
        }
        if ( memchr(data, '\0', 1024) != NULL ) {
            printf("binary data\n");
        } else {
            printf("ascii data\n");
        }
    }

    fclose(fd);

    return(0);
}
