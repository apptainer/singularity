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
#include <sys/mount.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <sys/param.h>
#include <errno.h> 
#include <signal.h>
#include <string.h>
#include <fcntl.h>  
#include <grp.h>
#include "config.h"

#ifndef LIBEXECDIR
#define LIBEXECDIR "undefined"
#endif

void sighandler(int sig) {
    signal(sig, sighandler);

    printf("Caught signal: %d\n", sig);
    fflush(stdout);
}


int main(int argc, char **argv) {
    char *sappdir;
    char *singularitypath;
    char *devpath;
    char *procpath;
    struct stat sappdirstat;
    struct stat singularitystat;
    int cwd_fd;
    int opt_contain = 0;
    int retval = 0;
    pid_t child_pid;
    mode_t process_mask = umask(0);
    uid_t uid = getuid();
    gid_t gid = getgid();


    signal(SIGINT, sighandler);

    /*
     * Prep work
     */


    // We don't run as root!
    if ( uid == 0 || gid == 0 ) {
        fprintf(stderr, "ERROR: Do not run singularities as root!\n");
        return(255);
    }

    // Lets start off as the right user.
    if ( seteuid(uid) != 0 ) {
        fprintf(stderr, "ERROR: Could not set effective user privledges to %d!\n", uid);
        return(255);
    }

    // Get sappdir from the environment (we check on this shortly)
    sappdir = getenv("SAPPCONTAINER");

    // Check for SINGULARITY_CONTAIN environment variable
    if ( getenv("SINGULARITY_CONTAIN") != NULL ) {
        opt_contain = 1;
    }

    // Open a FD to the current working dir.
    if ( (cwd_fd = open(".", O_RDONLY)) < 0 ) {
        fprintf(stderr, "ERROR: Could not open cwd fd (%s)!\n", strerror(errno));
        return(1);
    }


    /*
     * Sanity Checks, exit if any don't match.
     */

    // Make sure SAPPCONTAINER is defined
    if ( sappdir == NULL ) {
        fprintf(stderr, "ERROR: SAPPCONTAINER undefined!\n");
        return(1);
    }

    // Check SAPPCONTAINER
    if (lstat(sappdir, &sappdirstat) < 0) {
        fprintf(stderr, "ERROR: Could not stat %s!\n", sappdir);
        return(1);
    }
    if ( ! S_ISDIR(sappdirstat.st_mode) ) {
        fprintf(stderr, "ERROR: SAPPCONTAINER (%s) must be a SAPP directory!\n", sappdir);
        return(1);
    }
    if ( uid != (int)sappdirstat.st_uid ) {
        fprintf(stderr, "ERROR: Will not execute in a SAPPCONTAINER you don't own. (%s:%d)!\n", sappdir, (int)sappdirstat.st_uid);
        return(255);
    }
    
    // Check the singularity within the SAPPCONTAINER
    singularitypath = (char *) malloc(strlen(sappdir) + 13);
    snprintf(singularitypath, strlen(sappdir) + 13, "%s/singularity", sappdir);
    if ( stat(singularitypath, &singularitystat) < 0 ) {
        fprintf(stderr, "ERROR: Could not stat %s!\n", singularitypath);
        return(1);
    }
    if ( ! S_ISREG(singularitystat.st_mode) ) {
        fprintf(stderr, "ERROR: The singularity is not found in SAPPCONTAINER!\n");
        return(1);
    }
    if ( (int)singularitystat.st_uid != uid ) {
        fprintf(stderr, "ERROR: Will not execute a singularity you don't own. (%d)!\n", (int)sappdirstat.st_uid);
        return(255);
    }
    if ( ! (S_IXUSR & singularitystat.st_mode) ) {
        fprintf(stderr, "ERROR: The singularity can not be executed!\n");
        return(1);
    }

    // Populate paths for bind mounts
    devpath = (char *) malloc(strlen(sappdir) + 5);
    snprintf(devpath, strlen(sappdir) + 5, "%s/dev", sappdir);
    procpath = (char *) malloc(strlen(sappdir) + 6);
    snprintf(procpath, strlen(sappdir) + 6, "%s/proc", sappdir);


    // Create directories as neccessary
    if ( mkdir(procpath, S_IRUSR | S_IWUSR | S_IXUSR | S_IRGRP | S_IWGRP | S_IROTH | S_IXOTH) > 0 ) {
        fprintf(stderr, "ERROR: Could not create directory %s\n", procpath);
        return(255);
    }
    if ( mkdir(devpath, S_IRUSR | S_IWUSR | S_IXUSR | S_IRGRP | S_IWGRP | S_IROTH | S_IXOTH) > 0 ) {
        fprintf(stderr, "ERROR: Could not create directory %s\n", devpath);
        return(255);
    }

    umask(process_mask);


    // Entering danger zone
    if ( seteuid(0) != 0 ) {
        fprintf(stderr, "ERROR: Could not escalate effective user privledges!\n");
        return(255);
    }

    if ( mount(NULL, procpath, "proc", 0, NULL) != 0 ) {
        fprintf(stderr, "ERROR: Could not bind mount /proc\n");
        return(255);
    }
    if ( mount("/dev", devpath, NULL, MS_BIND, NULL) != 0 ) {
        fprintf(stderr, "ERROR: Could not bind mount /dev\n");
        return(255);
    }

    child_pid = fork();

    if ( child_pid == 0 ) {

        // Do the chroot
        if ( chroot(sappdir) != 0 ) {
            fprintf(stderr, "ERROR: failed enter SAPPCONTAINER: %s\n", sappdir);
            return(255);
        }

        // Dump all privs
        if ( setregid(gid, gid) != 0 ) {
            fprintf(stderr, "ERROR: Could not dump real/effective group privledges!\n");
            return(255);
        }
        if ( setreuid(uid, uid) != 0 ) {
            fprintf(stderr, "ERROR: Could not dump real/effective user privledges!\n");
            return(255);
        }

        // Confirm we no longer have any escalated privledges
        if ( setuid(0) == 0 ) {
            fprintf(stderr, "ERROR: Root not allowed here!\n");
            return(1);
        }

        // change directory back to starting point if needed
        if ( opt_contain > 0 ) {
            if ( fchdir(cwd_fd) != 0 ) {
                fprintf(stderr, "ERROR: Could not fchdir!\n");
                return(1);
            }
        else
            if ( chdir("/") != 0 ) {
                fprintf(stderr, "ERROR: Could not changedir to /\n");
                return(1);
            }
        }

        // Exec the singularity
        if ( execv("/singularity", argv) != 0 ) {
            fprintf(stderr, "ERROR: Failed to exec SAPP envrionment\n");
            return(2);
        }

    } else if ( child_pid > 0 ) {
        waitpid(child_pid, &retval, 0);
    } else {
        fprintf(stderr, "ERROR: Could not fork child process\n");
        retval++;
    }


    if ( umount(devpath) != 0 ) {
        fprintf(stderr, "ERROR: Could not unmount %s\n", devpath);
        retval++;
    }
    if ( umount(procpath) != 0 ) {
        fprintf(stderr, "ERROR: Could not unmount %s\n", procpath);
        retval++;
    }

    if ( close(cwd_fd) != 0 ) {
        fprintf(stderr, "ERROR: Could not close cwd_fd!\n");
        retval++;
    }

    return(retval);
}
