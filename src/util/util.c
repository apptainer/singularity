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
#include <linux/limits.h>
#include <ctype.h>

#include "config.h"
#include "util/util.h"
#include "lib/message.h"


char *envar(char *name, char *allowed, int len) {
    char *ret;
    char *env = getenv(name); // Flawfinder: ignore
    int count;

    singularity_message(VERBOSE2, "Checking input from environment: '%s'\n", name);

    singularity_message(DEBUG, "Checking environment variable is defined: %s\n", name);
    if ( env == NULL ) {
        singularity_message(VERBOSE2, "Environment variable is NULL: %s\n", name);
        return(NULL);
    }

    singularity_message(DEBUG, "Checking environment variable length (<= %d): %s\n", len, name);
    if ( strlength(env, len+1) > len) {
        singularity_message(ERROR, "Input length of '%s' is larger then allowed: %d\n", name, len);
        ABORT(255);
    }

    singularity_message(DEBUG, "Checking environment variable has allowed characters: %s\n", name);
    ret = (char *) malloc(len+1);
    for(count=0; count <= len && env[count] != '\0'; count++) {
        int test_char = env[count];
        int c, success = 0;
        if ( isalnum(test_char) > 0 ) {
            success = 1;
        } else {
            for (c=0; allowed[c] != '\0'; c++) {
                if ( test_char == allowed[c] ) {
                    success = 1;
                    continue;
                }
            }
        }
        if ( success == 0 ) {
            singularity_message(ERROR, "Illegal input character '%c' in: '%s=%s'\n", test_char, name, env);
            ABORT(255);
        }
        ret[count] = test_char;
    }
    ret[count] = '\0';

    singularity_message(VERBOSE2, "Obtained input from environment '%s' = '%s'\n", name, ret);
    return(ret);
}

int envar_defined(char *name) {
    singularity_message(DEBUG, "Checking if environment variable is defined: %s\n", name);
    if ( getenv(name) == NULL ) { // Flawfinder: ignore
        singularity_message(VERBOSE2, "Environment variable is undefined: %s\n", name);
        return(FALSE);
    }
    singularity_message(VERBOSE2, "Environment variable is defined: %s\n", name);
    return(TRUE);
}

char *envar_path(char *name) {
    singularity_message(DEBUG, "Checking environment variable is valid path: '%s'\n", name);
    return(envar(name, "/._-=,:", PATH_MAX));
}


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

char *joinpath(const char * path1, const char * path2) {
    char *tmp_path1 = strdup(path1);
    int path1_len = strlength(tmp_path1, 4096);
    char *ret;

    if ( tmp_path1[path1_len - 1] == '/' ) {
        tmp_path1[path1_len - 1] = '\0';
    }
    if ( path2[0] == '/' ) {
        path2++;
    }

    size_t ret_pathlen = strlength(tmp_path1, PATH_MAX) + strlength(path2, PATH_MAX) + 2;
    ret = (char *) malloc(ret_pathlen);
    if (snprintf(ret, ret_pathlen, "%s/%s", tmp_path1, path2) >= ret_pathlen) { // Flawfinder: ignore
        singularity_message(ERROR, "Overly-long path name.\n");
        ABORT(255);
    }

    return(ret);
}

char *strjoin(char *str1, char *str2) {
    char *ret;
    int len = strlength(str1, 2048) + strlength(str2, 2048) + 1;

    ret = (char *) malloc(len);
    if (snprintf(ret, len, "%s%s", str1, str2) >= len) { // Flawfinder: ignore
       singularity_message(ERROR, "Overly-long string encountered.\n");
       ABORT(255);
    }

    return(ret);
}

void chomp(char *str) {
    int len;
    int i;
    
    len = strlength(str, 4096);

    while ( str[0] == ' ' ) {
        for ( i = 1; i < len; i++ ) {
	    str[i-1] = str[i];
	}
	str[len] = '\0';
	len--;
    }

    while ( str[len - 1] == ' ' ) {
        str[len - 1] = '\0';
	len--;
    }

    if ( str[0] == '\n' ) {
        str[0] = '\0';
    }

    if ( str[len - 1] == '\n' ) {
        str[len - 1] = '\0';
    }
}

int strlength(const char *string, int max_len) {
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

int str2int(const char *input_str, long int *output_num) {
    long int result;
    char *endptr;
    errno = 0;
    // Empty string is an error:
    if ( *input_str == '\0' ) {
        errno = EINVAL;
        return -1;
    }

    result = strtol(input_str, &endptr, 10);
    // In the case of overflow / underflow or (possibly)
    // no digits consumed.
    if (errno) {return -1;}

    if ( *endptr == '\0' ) { // All data was consumed.
        if (output_num) {*output_num = result;}
        return 0;
    }
    errno = EINVAL;
    return -1;
}
