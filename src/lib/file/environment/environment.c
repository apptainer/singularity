/* 
 * Copyright (c) 2016, Michael W. Bauer. All rights reserved.
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
#include <limits.h>
#include <unistd.h>
#include <stdlib.h>
#include <dirent.h>

#include "util/file.h"
#include "util/util.h"
#include "lib/message.h"
#include "lib/file/environment/environment.h"


int singularity_file_environment() {
    struct dirent **namelist;
    char *meta_file = strdup("");
    char *buff_line;
    int rootfs_fd = singularity_rootfs_fd();
    int size = 1; //Starts with 1 to account for single null terminator
    int i;

    singularity_message(DEBUG, "Sorting through /.env/ folder and assembling ordered list of files to source\n");
    
    if ( scandirat(rootfs_fd, ".env/", &namelist, filter_metafile, compare_filenames) < 0 ) {
        return(-1);
    }
    
    for (i = 0; namelist[i] != NULL; i++) {
        buff_line = strjoin("source ", (namelist[i])->d_name);
        size = size + strlength(buff_line, 2048) + 1; //+1 for newline
        if ( (meta_file = (char *)realloc(meta_file, size)) == NULL ) {
            return(-1);
        }
        snprintf(meta_file, size, "%s\n%s", meta_file, buff_line);

        free(buff_line);
        free(namelist[i]);
    }

    fchdir(rootfs_fd);
    if ( ( i = fileput(".env/.metasource", meta_file) ) != 0 ) {
        singularity_message(WARNING, "Unable to write .metasource file: %s\n", strerror(errno));
    }

    free(meta_file);
    return(0);
}

int filter_metafile(const struct dirent *entry) {
    return( strncmp(entry->d_name, ".metasource", 11) );
}

int compare_filenames(const struct dirent **a, const struct dirent **b) {
    long int a_int, b_int;
    if ( (str2int((*a)->d_name, &a_int) != 0) || (str2int((*b)->d_name, &b_int) != 0) ) {
        return(-1);
    } else {
        return(a_int - b_int);
    }
}
