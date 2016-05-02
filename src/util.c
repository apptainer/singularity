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

#include "config.h"


int is_file(char *path) {
    struct stat filestat;

    // Stat path
    if (lstat(path, &filestat) < 0) {
        return(-1);
    }

    // Test path
    if ( S_ISREG(filestat.st_mode) ) {
        return(0);
    }

    return(-1);
}

int is_link(char *path) {
    struct stat filestat;

    // Stat path
    if (lstat(path, &filestat) < 0) {
        return(-1);
    }

    // Test path
    if ( S_ISLNK(filestat.st_mode) ) {
        return(0);
    }

    return(-1);
}

int is_dir(char *path) {
    struct stat filestat;

    // Stat path
    if (lstat(path, &filestat) < 0) {
        return(-1);
    }

    // Test path
    if ( S_ISDIR(filestat.st_mode) ) {
        return(0);
    }

    return(-1);
}

int is_exec(char *path) {
    struct stat filestat;

    // Stat path
    if (lstat(path, &filestat) < 0) {
        return(-1);
    }

    // Test path
    if ( (S_IXUSR & filestat.st_mode) ) {
        return(0);
    }

    return(-1);
}

int is_owner(char *path, uid_t uid) {
    struct stat filestat;

    // Stat path
    if (lstat(path, &filestat) < 0) {
        return(-1);
    }

    if ( uid == (int)filestat.st_uid ) {
        return(0);
    }

    return(-1);
}

int s_mkpath(char *dir, mode_t mode) {
    if (!dir) {
        return(-1);
    }

    if (strlen(dir) == 1 && dir[0] == '/') {
        return(0);
    }

    if ( is_dir(dir) == 0 ) {
        // Directory already exists, stop...
        return(0);
    }

    s_mkpath(dirname(strdupa(dir)), mode);

    if ( mkdir(dir, mode) < 0 ) {
        printf("ERROR: Could not create directory %s: %s\n", dir, strerror(errno));
        return(-1);
    }

    return(0);
}

// TODO: This needs to remove only until a second path argument
int s_rmdir(char *dir) {
    if (!dir) {
        return(-1);
    }

    if (strlen(dir) == 1 && dir[0] == '/') {
        return(0);
    }

    s_rmdir(dirname(strdupa(dir)));

    return unlink(dir);
}

int intlen(int input) {
    unsigned int len = 1;

    while (input /= 10) {
        len ++;
    }

    return(len);
}


int copy_file(char * source, char * dest) {
    char c;
    FILE * fd_s;
    FILE * fd_d;

    if ( is_file(source) < 0 ) {
        printf("No such file: %s->%s\n", source, dest);
        return(-1);
    }

    fd_s = fopen(source, "r");
    if ( fd_s == NULL ) {
        fprintf(stderr, "ERROR: Could not read %s: %s\n", source, strerror(errno));
        return(-1);
    }

    fd_d = fopen(dest, "w");
    if ( fd_s == NULL ) {
        fclose(fd_s);
        fprintf(stderr, "ERROR: Could not write %s: %s\n", dest, strerror(errno));
        return(-1);
    }

    while ( ( c = fgetc(fd_s) ) != EOF ) {
        fputc(c, fd_d);
    }

    fclose(fd_s);
    fclose(fd_d);

    return(0);
}


char * joinpath(char * path1, char * path2) {
    char *ret;

    ret = (char *) malloc(strlen(path1) + strlen(path2) + 2);
    snprintf(ret, strlen(path1) + strlen(path2) + 2, "%s/%s", path1, path2);

    return(ret);
}
