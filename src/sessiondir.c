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

#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <string.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/file.h>
#include <sys/wait.h>
#include <unistd.h>
#include <stdlib.h>
#include <limits.h>



#include "file.h"
#include "image.h"
#include "util.h"
#include "message.h"
#include "privilege.h"
#include "config_parser.h"
#include "fork.h"


char *sessiondir = NULL;
int sessiondir_fd = 0;

char *singularity_sessiondir_init(char *file) {
    pid_t child_pid;
    int retval;


    if ( file == NULL ) {
        message(DEBUG, "Got null for file, returning prior sessiondir\n");
    } else {
        char *sessiondir_prefix;
        struct stat filestat;
        uid_t uid = singularity_priv_getuid();

        sessiondir = (char *) malloc(sizeof(char) * PATH_MAX);

        message(DEBUG, "Checking Singularity configuration for 'sessiondir prefix'\n");

        if (stat(file, &filestat) < 0) {
            message(ERROR, "Failed calling stat() on %s: %s\n", file, strerror(errno));
            return(NULL);
        }

        config_rewind();
        if ( ( sessiondir_prefix = config_get_key_value("sessiondir prefix") ) != NULL ) {
            snprintf(sessiondir, sizeof(char) * PATH_MAX, "%s%d.%d.%lu", sessiondir_prefix, (int)uid, (int)filestat.st_dev, (long unsigned)filestat.st_ino);
        } else {
            snprintf(sessiondir, sizeof(char) * PATH_MAX, "/tmp/.singularity-session-%d.%d.%lu", (int)uid, (int)filestat.st_dev, (long unsigned)filestat.st_ino);
        }
        message(DEBUG, "Set sessiondir to: %s\n", sessiondir);
    }

    if ( is_dir(sessiondir) < 0 ) {
        if ( s_mkpath(sessiondir, 0755) < 0 ) {
            message(ERROR, "Failed creating session directory\n");
            ABORT(255);
        }
    }

    message(DEBUG, "Opening sessiondir file descriptor\n");
    if ( ( sessiondir_fd = open(sessiondir, O_RDONLY) ) < 0 ) {
        message(ERROR, "Could not obtain file descriptor for session directory %s: %s\n", sessiondir, strerror(errno));
        ABORT(255);
    }

    message(DEBUG, "Setting shared flock() on session directory\n");
    if ( flock(sessiondir_fd, LOCK_SH | LOCK_NB) < 0 ) {
        message(ERROR, "Could not obtain shared lock on %s: %s\n", sessiondir, strerror(errno));
        ABORT(255);
    }

    if ( ( child_pid = singularity_fork() ) > 0 ) {
        int tmpstatus;

        message(DEBUG, "Waiting on NS child process\n");

        waitpid(child_pid, &tmpstatus, 0);
        retval = WEXITSTATUS(tmpstatus);

        message(DEBUG, "Checking to see if we are the last process running in this sessiondir\n");
        if ( flock(sessiondir_fd, LOCK_EX | LOCK_NB) == 0 ) {
            message(VERBOSE, "Cleaning sessiondir: %s\n", sessiondir);
            if ( s_rmdir(sessiondir) < 0 ) {
                message(ERROR, "Could not remove session directory\n");
                ABORT(255);
            }
        }
    
        exit(retval);
    }

    return(sessiondir);
}

char *singularity_sessiondir_get(void) {
    if ( sessiondir == NULL ) {
        message(ERROR, "Doh, session directory has not been setup!\n");
        ABORT(255);
    }
    message(DEBUG, "Returning: %s\n", sessiondir);
    return(sessiondir);
}

int singularity_sessiondir_rm(void) {
    if ( sessiondir == NULL ) {
        message(ERROR, "Session directory is NULL, can not remove nullness!\n");
        return(-1);
    }

    message(DEBUG, "Checking to see if we are the last process running in this sessiondir\n");
    if ( flock(sessiondir_fd, LOCK_EX | LOCK_NB) == 0 ) {
        message(VERBOSE, "Cleaning sessiondir: %s\n", sessiondir);
        if ( s_rmdir(sessiondir) < 0 ) {
            message(ERROR, "Could not remove session directory\n");
            ABORT(255);
        }
    }
    
    return(0);
}
