/*
 *
 * Copyright (c) 2015-2016, Gregory M. Kurtzer
 * All rights reserved.
 *
 *
 * Copyright (c) 2015-2016, The Regents of the University of California,
 * through Lawrence Berkeley National Laboratory (subject to receipt of
 * any required approvals from the U.S. Dept. of Energy).
 * All rights reserved.
 *
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
#include "config.h"



int s_is_file(char *path) {
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

int s_is_dir(char *path) {
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

int s_is_exec(char *path) {
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

int s_is_owner(char *path, uid_t uid) {
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

    s_mkpath(dirname(strdupa(dir)), mode);

    return mkdir(dir, mode);
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
