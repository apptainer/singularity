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
#include "util.h"


char *file_id(char *path) {
    struct stat filestat;
    char *ret;
    uid_t uid = getuid();

    // Stat path
    if (lstat(path, &filestat) < 0) {
        return(NULL);
    }

    ret = (char *) malloc(128);
    snprintf(ret, 128, "%d.%d.%lu", (int)uid, (int)filestat.st_dev, (long unsigned)filestat.st_ino);
    return(ret);
}


int is_file(char *path) {
    struct stat filestat;

    // Stat path
    if (stat(path, &filestat) < 0) {
        return(-1);
    }

    // Test path
    if ( S_ISREG(filestat.st_mode) ) {
        return(0);
    }

    return(-1);
}

int is_fifo(char *path) {
    struct stat filestat;

    // Stat path
    if (stat(path, &filestat) < 0) {
        return(-1);
    }

    // Test path
    if ( S_ISFIFO(filestat.st_mode) ) {
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

int is_blk(char *path) {
    struct stat filestat;

    // Stat path
    if (lstat(path, &filestat) < 0) {
        return(-1);
    }

    // Test path
    if ( S_ISBLK(filestat.st_mode) ) {
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

    if ( s_mkpath(dirname(strdupa(dir)), mode) < 0 ) {
        // Return if priors failed
        return(-1);
    }

    if ( mkdir(dir, mode) < 0 ) {
        return(-1);
    }

    return(0);
}

int _unlink(const char *fpath, const struct stat *sb, int typeflag, struct FTW *ftwbuf) {
//    printf("remove(%s)\n", fpath);
    return(remove(fpath));
}

int s_rmdir(char *dir) {
    return(nftw(dir, _unlink, 32, FTW_DEPTH));
}

int copy_file(char * source, char * dest) {
    int c;
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


int fileput(char *path, char *string) {
    FILE *fd;

    fd = fopen(path, "w");
    if ( fd == NULL ) {
        fprintf(stderr, "ERROR: Could not write to %s: %s\n", path, strerror(errno));
        return(-1);
    }

    fprintf(fd, "%s", string);
    fclose(fd);

    return(0);
}

char *filecat(char *path) {
    char *ret;
    FILE *fd;
    int c;
    long length;
    long pos = 0;
    
    if ( is_file(path) < 0 ) {
        fprintf(stderr, "ERROR: Could not find %s\n", path);
        return(NULL);
    }

    fd = fopen(path, "r");
    if ( fd == NULL ) {
        fprintf(stderr, "ERROR: Could not read from %s: %s\n", path, strerror(errno));
        return(NULL);
    }


    if ( fseek(fd, 0L, SEEK_END) < 0 ) {
        fprintf(stderr, "ERROR: Could not seek to end of file %s: %s\n", path, strerror(errno));
        return(NULL);
    }

    length = ftell(fd);

    rewind(fd);

    ret = (char *) malloc(length+1);

    while ( ( c = fgetc(fd) ) != EOF ) {
        ret[pos] = c;
        pos++;
    }
    ret[pos] = '\0';

    fclose(fd);

    return(ret);
}

char * container_dir_walk(char *containerdir, char *dir) {
    char * testdir = strdup(dir);
    char * prevdir = NULL;
    if ( containerdir == NULL || dir == NULL ) {
        return(NULL);
    }

    while ( testdir != NULL && ( strcmp(testdir, "/") != 0 ) ) {
        if ( is_dir(joinpath(containerdir, testdir)) == 0 ) {
            return(testdir);
        }
        prevdir = strdup(testdir);
        testdir = dirname(strdup(testdir));
    }
    return(prevdir);
}

