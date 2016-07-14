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


#define _XOPEN_SOURCE 500 // For nftw
#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/mount.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <errno.h> 
#include <string.h>
#include <fcntl.h>  
#include <libgen.h>
#include <assert.h>
#include <ftw.h>
#include <time.h>

#include "config.h"
#include "message.h"
#include "util.h"


int intlen(int input) {
    unsigned int len = 1;

    while (input /= 10) {
        len ++;
    }

    return(len);
}

char *int2str(int num) {
    char *ret;
    
    ret = (char *) malloc(intlen(num) + 1);

    snprintf(ret, intlen(num) + 1, "%d", num); // Flawfinder: ignore

    return(ret);
}

char *joinpath(char * path1, char * path2) {
    char *ret;

    ret = (char *) malloc(strlength(path1, 2048) + strlength(path2, 2048) + 2);
    snprintf(ret, strlen(path1) + strlen(path2) + 2, "%s/%s", path1, path2); // Flawfinder: ignore

    return(ret);
}

char *strjoin(char *str1, char *str2) {
    char *ret;
    int len = strlength(str1, 2048) + strlength(str2, 2048) + 1;

    ret = (char *) malloc(len);
    snprintf(ret, len, "%s%s", str1, str2); // Flawfinder: ignore

    return(ret);
}

void chomp(char *str) {
    int len = strlength(str, 4096);
    if ( str[len - 1] == ' ') {
        str[len - 1] = '\0';
    }
    if ( str[0] == '\n') {
        str[0] = '\0';
    }
    if ( str[len - 1] == '\n') {
        str[len - 1] = '\0';
    }
}

int strlength(char *string, int max_len) {
    int len;
    for (len=0; string[len] && len < max_len; len++) {
        // Do nothing in the loop
    }
    return(len);
}

/*
char *random_string(int length) {
    static const char characters[] = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
    char *ret;
    int i;
    int pid = getpid();

    ret = (char *) malloc(length);
 
    srand(time(NULL) * pid);
    for (i = 0; i < length; ++i) {
        ret[i] = characters[rand() % (sizeof(characters) - 1)];
    }
 
    ret[length] = '\0';

    return(ret);
}
*/
