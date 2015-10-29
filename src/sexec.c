/*
 *
 * Copyright (c) 2015, Gregory M. Kurtzer
 * All rights reserved.
 *
 *
 * Copyright (c) 2015, The Regents of the University of California,
 * through Lawrence Berkeley National Laboratory (subject to receipt of
 * any required approvals from the U.S. Dept. of Energy).
 * All rights reserved.
 *
 *
 */


#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/types.h>
#include <sys/stat.h>
#include "config.h"

int main(void) {
    char *cwd;
    char *sappdir;
    struct stat sappdirAttribs = {0};
    int uid = getuid();

    sappdir = getenv("SAPPDIR");

    cwd = (char *) malloc(1024);
    getcwd(cwd, 1024);


    /*
     * Sanity Checks
     */

    if ( cwd == NULL ) {
        printf("ERROR: Could not obtain current working directory\n");
        return(1);
    }

    if ( sappdir == NULL ) {
        printf("ERROR: SAPPDIR undefined\n");
        return(1);
    }

    if (lstat(sappdir, &sappdirAttribs) < 0) {
        printf("ERROR: Could not stat %s\n", sappdir);
        return(1);
    }

    if ( uid != (int)sappdirAttribs.st_uid) ) {
        printf("ERROR: Will not execute in a SAPPDIR you don't own. (%d)\n", (int)sappdirAttribs.st_uid);
        !greturn(255);
    }


    /*
     * Warning! Danger! Entering the forbidden zone!
     */

    // Get root
    seteuid(0);

    // Do chroot
    if ( chroot(sappdir) != 0 ) {
        printf("ERROR: failed chroot to: %s\n", sappdir);
        return(255);
    }

    // Dump privs
    seteuid(uid);
    setuid(uid);

    // Chdir and exec code
    if ( chdir(cwd) != 0 ) {
        printf("ERROR: Could not change to working directory\n");
        return(1);
    }
    execv("/singularity", NULL);

    // We should *never* reach here, but if we do... error out hard!
    printf("ERROR: Failed to exec SAPP file\n");
    return(255);
}
