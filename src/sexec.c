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
#include <sys/param.h>
#include <errno.h> 
#include <string.h>
#include <fcntl.h>  
#include <grp.h>
#include "config.h"

int main(int argc, char **argv) {
    char *sappdir;
    struct stat sappdirAttribs = {0};
    uid_t uid = getuid();
    gid_t gid = getgid();
    int cwd_fd;

    sappdir = getenv("SAPPDIR");

    /*
     * Open a FD to the current working dir.
     */

    if ( (cwd_fd = open(".", O_RDONLY)) < 0 ) {
        fprintf(stderr, "ERROR: Could not open cwd fd (%s)!\n", strerror(errno));
        return(1);
    }


    /*
     * Sanity Checks, exit if any don't match.
     */

    if ( sappdir == NULL ) {
        fprintf(stderr, "ERROR: SAPPDIR undefined!\n");
        return(1);
    }

    if (lstat(sappdir, &sappdirAttribs) < 0) {
        fprintf(stderr, "ERROR: Could not stat %s!\n", sappdir);
        return(1);
    }

    if ( uid != (int)sappdirAttribs.st_uid ) {
        fprintf(stderr, "ERROR: Will not execute in a SAPPDIR you don't own. (%s:%d)!\n", sappdir, (int)sappdirAttribs.st_uid);
        return(255);
    }


    /*
     * Warning! Danger! Entering the privledged zone!
     */

    // Get root
    if ( seteuid(0) != 0 ) {
        fprintf(stderr, "ERROR: Could not escalate privledges!\n");
        return(1);
    }

    // Do chroot before dropping privs to escape chroot
    if ( chroot(sappdir) != 0 ) {
        fprintf(stderr, "ERROR: failed enter SAPPDIR: %s\n", sappdir);
        return(255);
    }

    // Dump privs
    if ( setregid(gid, gid) != 0 ) {
        fprintf(stderr, "ERROR: Could not dump real/effective group privledges!\n");
        return(255);
    }
    if ( setreuid(uid, uid) != 0 ) {
        fprintf(stderr, "ERROR: Could not dump real/effective user privledges!\n");
        return(255);
    }


    /*
     * Out of the immediate danger zone... whew!
     */

    // Confirm we no longer have any escalated privledges
    if ( setuid(0) == 0 ) {
        fprintf(stderr, "ERROR: Root not allowed here!\n");
        return(1);
    }

    // change directory back to starting point
    if ( fchdir(cwd_fd) != 0 ) {
        fprintf(stderr, "ERROR: Could not fchdir!\n");
        return(1);
    }
    if ( close(cwd_fd) != 0 ) {
        fprintf(stderr, "ERROR: Could not close cwd_fd!\n");
        return(1);
    }

    // Exec 
    if ( execv("/singularity", argv) != 0 ) {
    //if ( execv("/singularity", (char **) "") != 0 ) {
        fprintf(stderr, "ERROR: Failed to exec SAPP envrionment\n");
        return(2);
    }

    // We should *never* reach here, but if we do... error out hard!
    fprintf(stderr, "ERROR: We should not be here!\n");
    return(255);
}
