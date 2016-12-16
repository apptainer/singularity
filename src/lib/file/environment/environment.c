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

#define _GNU_SOURCE
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
#include "lib/singularity.h"
#include "lib/file/environment/environment.h"


int singularity_file_environment(void) {
    struct dirent **namelist;
    char *meta_file = strdup("");
    char *buff_line;
    char *buff_line2;
    int rootfs_fd = singularity_rootfs_fd();
    int size = 1; //Starts with 1 to account for single null terminator
    int i;
    int ret = 0;

    singularity_message(DEBUG, "Sorting through /.env/ folder and assembling ordered list of files to source\n");
    
    if ( ( ret = scandirat(rootfs_fd, ".env/", &namelist, filter_metafile, compare_filenames) ) < 0 ) {
        return(-1);
    } else if ( ret == 0 ) {
        singularity_message(DEBUG, "No files in /.env/, adding empty file\n");
    }
    
    for (i = 0; i < ret; i++) {
        buff_line = strjoin("source /.env/", (namelist[i])->d_name);
        size = size + strlength(buff_line, 2048) + 1; //+1 for newline
        if ( (meta_file = (char *)realloc(meta_file, size)) == NULL ) {
            singularity_message(ERROR, "Memory allocation failed: %s\n", strerror(errno));
            return(-1);
        }
        buff_line2 = strdup(meta_file);
        snprintf(meta_file, size, "%s\n%s", buff_line2, buff_line);

        free(buff_line2);
        free(buff_line);
        free(namelist[i]);
    }

    singularity_message(DEBUG, "Writing to /.env/.metafile:%s\n", meta_file);

    if ( ( i = fileputat(rootfs_fd, ".env/.metasource", meta_file) ) != 0 ) {
        singularity_message(DEBUG, "Unable to write .metasource file: %s\n", strerror(errno));
    }

    free(namelist);
    free(meta_file);
    return(0);
}

int filter_metafile(const struct dirent *entry) {
    return( strncmp(entry->d_name, ".", 1) );
}

int compare_filenames(const struct dirent **a, const struct dirent **b) {
    long int a_int, b_int;
    if ( (str2int((*a)->d_name, &a_int) != 0) || (str2int((*b)->d_name, &b_int) != 0) ) {
        return(-1);
    } else {
        return(a_int - b_int);
    }
}
