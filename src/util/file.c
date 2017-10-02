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
#include "util/util.h"
#include "util/message.h"
#include "util/privilege.h"

char *file_id(char *path) {
    struct stat filestat;
    char *ret;
    uid_t uid = singularity_priv_getuid();

    singularity_message(DEBUG, "Called file_id(%s)\n", path);

    // Stat path
    if (lstat(path, &filestat) < 0) {
        return(NULL);
    }

    ret = (char *) malloc(128);
    snprintf(ret, 128, "%d.%d.%lu", (int)uid, (int)filestat.st_dev, (long unsigned)filestat.st_ino); // Flawfinder: ignore

    singularity_message(VERBOSE2, "Generated file_id: %s\n", ret);

    singularity_message(DEBUG, "Returning file_id(%s) = %s\n", path, ret);
    return(ret);
}

char *file_devino(char *path) {
    struct stat filestat;
    char *ret;

    singularity_message(DEBUG, "Called file_devino(%s)\n", path);

    // Stat path
    if (lstat(path, &filestat) < 0) {
        return(NULL);
    }

    ret = (char *) malloc(128);
    snprintf(ret, 128, "%d.%lu", (int)filestat.st_dev, (long unsigned)filestat.st_ino); // Flawfinder: ignore

    singularity_message(DEBUG, "Returning file_devino(%s) = %s\n", path, ret);
    return(ret);
}

int chk_perms(char *path, mode_t mode) {
    struct stat filestat;

    singularity_message(DEBUG, "Checking permissions on: %s\n", path);

    // Stat path
    if (stat(path, &filestat) < 0) {
        return(-1);
    }

    if ( filestat.st_mode & mode ) {
        singularity_message(WARNING, "Found appropriate permissions on file: %s\n", path);
        return(0);
    }

    return(-1);
}

int chk_mode(char *path, mode_t mode, mode_t mask) {
    struct stat filestat;

    singularity_message(DEBUG, "Checking exact mode (%o) on: %s\n", mode, path);

    // Stat path
    if (stat(path, &filestat) < 0) {
        return(-1);
    }

    if ( ( filestat.st_mode | mask )  == ( mode | mask ) ) {
        singularity_message(DEBUG, "Found appropriate mode on file: %s\n", path);
        return(0);
    } else {
        singularity_message(VERBOSE, "Found wrong permission on file %s: %o != %o\n", path, mode, filestat.st_mode);
    }

    return(-1);
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
    if (stat(path, &filestat) < 0) {
        return(-1);
    }

    // Test path
    if ( S_ISDIR(filestat.st_mode) ) {
        return(0);
    }

    return(-1);
}

int is_suid(char *path) {
    struct stat filestat;

    // Stat path
    if (stat(path, &filestat) < 0) {
        return(-1);
    }

    // Test path
    if ( (S_ISUID & filestat.st_mode) ) {
        return(0);
    }

    return(-1);
}

int is_exec(char *path) {
    struct stat filestat;

    // Stat path
    if (stat(path, &filestat) < 0) {
        return(-1);
    }

    // Test path
    if ( (S_IXUSR & filestat.st_mode) ) {
        return(0);
    }

    return(-1);
}

int is_write(char *path) {
    struct stat filestat;

    // Stat path
    if (stat(path, &filestat) < 0) {
        return(-1);
    }

    // Test path
    if ( (S_IWUSR & filestat.st_mode) ) {
        return(0);
    }

    return(-1);
}

int is_owner(char *path, uid_t uid) {
    struct stat filestat;

    // Stat path
    if (stat(path, &filestat) < 0) {
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
    if (stat(path, &filestat) < 0) {
        return(-1);
    }

    // Test path
    if ( S_ISBLK(filestat.st_mode) ) {
        return(0);
    }

    return(-1);
}


int is_chr(char *path) {
    struct stat filestat;

    // Stat path
    if (stat(path, &filestat) < 0) {
        return(-1);
    }

    // Test path
    if ( S_ISCHR(filestat.st_mode) ) {
        return(0);
    }

    return(-1);
}


int s_mkpath(char *dir, mode_t mode) {
    if (!dir) {
        return(-1);
    }

    if (strcmp(dir, "/") == 0 ) {
        singularity_message(DEBUG, "Directory is '/', returning '0'\n");
        return(0);
    }

    if ( is_dir(dir) == 0 ) {
        singularity_message(DEBUG, "Directory exists, returning '0': %s\n", dir);
        return(0);
    }

    if ( is_dir(dirname(strdupa(dir))) < 0 ) {
        singularity_message(DEBUG, "Creating parent directory: %s\n", dirname(strdupa(dir)));
        if ( s_mkpath(dirname(strdupa(dir)), mode) < 0 ) {
            singularity_message(VERBOSE, "Failed to create parent directory %s\n", dir);
            return(-1);
        }
    }

    singularity_message(DEBUG, "Creating directory: %s\n", dir);
    mode_t mask = umask(0); // Flawfinder: ignore
    int ret = mkdir(dir, mode);
    umask(mask); // Flawfinder: ignore

    if ( ret < 0 ) {
        if ( errno != EEXIST ) {
            singularity_message(DEBUG, "Opps, could not create directory %s: (%d) %s\n", dir, errno, strerror(errno));
            return(-1);
        }
    }

    return(0);
}

int _unlink(const char *fpath, const struct stat *sb, int typeflag, struct FTW *ftwbuf) {
    int retval;

    if ( ( retval = remove(fpath) ) < 0 ) { 
        singularity_message(WARNING, "Failed removing file: %s\n", fpath);
    }

    return(retval);
}

int _writable(const char *fpath, const struct stat *sb, int typeflag, struct FTW *ftwbuf) {
    int retval;

    if ( is_link((char *) fpath) == 0 ) {
        return(0);
    }

    if ( ( retval = chmod(fpath, 0700) ) < 0 ) { // Flawfinder: ignore
        singularity_message(WARNING, "Failed changing permission of file: %s\n", fpath);
    }

    // Always return success
    return(0);
}

int s_rmdir(char *dir) {

    singularity_message(DEBUG, "Removing directory: %s\n", dir);
    if ( nftw(dir, _writable, 32, FTW_MOUNT|FTW_PHYS) < 0 ) {
        singularity_message(ERROR, "Failed preparing directory for removal: %s\n", dir);
        ABORT(255);
    }

    return(nftw(dir, _unlink, 32, FTW_DEPTH|FTW_MOUNT|FTW_PHYS));
}

int copy_file(char * source, char * dest) {
    struct stat filestat;
    int c;
    FILE * fp_s;
    FILE * fp_d;

    singularity_message(DEBUG, "Called copy_file(%s, %s)\n", source, dest);

    if ( is_file(source) < 0 ) {
        singularity_message(ERROR, "Could not copy from non-existent source: %s\n", source);
        return(-1);
    }

    singularity_message(DEBUG, "Opening source file: %s\n", source);
    if ( ( fp_s = fopen(source, "r") ) == NULL ) { // Flawfinder: ignore
        singularity_message(ERROR, "Could not read %s: %s\n", source, strerror(errno));
        return(-1);
    }

    singularity_message(DEBUG, "Opening destination file: %s\n", dest);
    if ( ( fp_d = fopen(dest, "w") ) == NULL ) { // Flawfinder: ignore
        fclose(fp_s);
        singularity_message(ERROR, "Could not write %s: %s\n", dest, strerror(errno));
        return(-1);
    }

    singularity_message(DEBUG, "Calling fstat() on source file descriptor: %d\n", fileno(fp_s));
    if ( fstat(fileno(fp_s), &filestat) < 0 ) {
        singularity_message(ERROR, "Could not fstat() on %s: %s\n", source, strerror(errno));
        fclose(fp_s);
        fclose(fp_d);
        return(-1);
    }

    singularity_message(DEBUG, "Cloning permission string of source to dest\n");
    if ( fchmod(fileno(fp_d), filestat.st_mode) < 0 ) {
        singularity_message(ERROR, "Could not set permission mode on %s: %s\n", dest, strerror(errno));
        fclose(fp_s);
        fclose(fp_d);
        return(-1);
    }

    singularity_message(DEBUG, "Copying file data...\n");
    while ( ( c = fgetc(fp_s) ) != EOF ) { // Flawfinder: ignore (checked boundries)
        fputc(c, fp_d);
    }

    singularity_message(DEBUG, "Done copying data, closing file pointers\n");
    fclose(fp_s);
    fclose(fp_d);

    singularity_message(DEBUG, "Returning copy_file(%s, %s) = 0\n", source, dest);

    return(0);
}


int fileput(char *path, char *string) {
    FILE *fd;

    singularity_message(DEBUG, "Called fileput(%s, %s)\n", path, string);
    if ( ( fd = fopen(path, "w") ) == NULL ) { // Flawfinder: ignore
        singularity_message(ERROR, "Could not write to %s: %s\n", path, strerror(errno));
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

    singularity_message(DEBUG, "Called filecat(%s)\n", path);
    
    if ( is_file(path) < 0 ) {
        singularity_message(ERROR, "Could not find %s\n", path);
        return(NULL);
    }

    if ( ( fd = fopen(path, "r") ) == NULL ) { // Flawfinder: ignore
        singularity_message(ERROR, "Could not read from %s: %s\n", path, strerror(errno));
        return(NULL);
    }


    if ( fseek(fd, 0L, SEEK_END) < 0 ) {
        singularity_message(ERROR, "Could not seek to end of file %s: %s\n", path, strerror(errno));
        fclose(fd);
        return(NULL);
    }

    length = ftell(fd);

    rewind(fd);

    ret = (char *) malloc(length+1);

    while ( ( c = fgetc(fd) ) != EOF ) { // Flawfinder: ignore (checked boundries)
        ret[pos] = c;
        pos++;
    }
    ret[pos] = '\0';

    fclose(fd);

    return(ret);
}

/* 
 * Open and exclusive-lock file, creating it (-rw-------)
 * if necessary. If fdptr is not NULL, the descriptor is
 * saved there. The descriptor is never one of the standard
 * descriptors STDIN_FILENO, STDOUT_FILENO, or STDERR_FILENO.
 * If successful, the function returns 0.
 * Otherwise, the function returns nonzero errno:
 *     EINVAL: Invalid lock file path
 *     EMFILE: Too many open files
 *     EALREADY: Already locked
 * or one of the open(2)/creat(2) errors.
 */
int filelock(const char *const filepath, int *const fdptr) {
    struct flock lock;
    int used = 0; /* Bits 0 to 2: stdin, stdout, stderr */
    int fd;

    singularity_message(DEBUG, "Called filelock(%s)\n", filepath);
    
    /* In case the caller is interested in the descriptor,
     * initialize it to -1 (invalid). */
    if (fdptr)
        *fdptr = -1;

    /* Invalid path? */
    if (filepath == NULL || *filepath == '\0')
        return errno = EINVAL;

    /* Open the file. */
    do {
        fd = open(filepath, O_RDWR | O_CREAT, 0644);
    } while (fd == -1 && errno == EINTR);
    if (fd == -1) {
        if (errno == EALREADY)
            errno = EIO;
        return errno;
    }

    /* Move fd away from the standard descriptors. */
    while (1) {
        if( fd == STDIN_FILENO ) {
            used |= 1;
            fd = dup(fd);
        } else if ( fd == STDOUT_FILENO ) {
            used |= 2;
            fd = dup(fd);
        } else if( fd == STDERR_FILENO ) {
            used |= 4;
            fd = dup(fd);
        } else {
            break;
        }
    }
    
    /* Close the standard descriptors we temporarily used. */
    if (used & 1)
        close(STDIN_FILENO);
    if (used & 2)
        close(STDOUT_FILENO);
    if (used & 4)
        close(STDERR_FILENO);

    /* Did we run out of descriptors? */
    if (fd == -1)
        return errno = EMFILE;    

    /* Exclusive lock, cover the entire file (regardless of size). */
    lock.l_type = F_WRLCK;
    lock.l_whence = SEEK_SET;
    lock.l_start = 0;
    lock.l_len = 0;
    if (fcntl(fd, F_SETLK, &lock) == -1) {
        /* Lock failed. Close file and report locking failure. */
        close(fd);
        return errno = EALREADY;
    }

    if ( fcntl(fd, F_SETFD, FD_CLOEXEC) != 0 ) {
        close(fd);
        return errno = EBADF;
    }

    /* Save descriptor, if the caller wants it. */
    if (fdptr)
        *fdptr = fd;

    return 0;
}


char *basedir(char *dir) {
    char *testdir = strdup(dir);
    char *ret = NULL;

    singularity_message(DEBUG, "Obtaining basedir for: %s\n", dir);

    while ( ( strcmp(testdir, "/") != 0 ) && ( strcmp(testdir, ".") != 0 ) ) {
        singularity_message(DEBUG, "Iterating basedir: %s\n", testdir);

        ret = strdup(testdir);
        testdir = dirname(strdup(testdir));
    }

    return(ret);
}




