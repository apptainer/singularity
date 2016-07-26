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

#define _GNU_SOURCE
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
#include <sched.h>
#include <string.h>
#include <fcntl.h>  
#include <grp.h>
#include <libgen.h>

#include "config.h"
#include "mounts.h"
#include "util.h"
#include "file.h"
#include "loop-control.h"
#include "message.h"


#ifndef LIBEXECDIR
#define LIBEXECDIR "undefined"
#endif
#ifndef SYSCONFDIR
#define SYSCONFDIR "/etc"
#endif
#ifndef LOCALSTATEDIR
#define LOCALSTATEDIR "/var/"
#endif

#ifndef MS_PRIVATE
#define MS_PRIVATE (1<<18)
#endif
#ifndef MS_REC
#define MS_REC 16384
#endif



pid_t namespace_fork_pid = 0;
pid_t exec_fork_pid = 0;



void sighandler(int sig) {
    signal(sig, sighandler);

    printf("Caught signal: %d\n", sig);
    fflush(stdout);

    if ( exec_fork_pid > 0 ) {
        fprintf(stderr, "Singularity is sending SIGKILL to child pid: %d\n", exec_fork_pid);

        kill(exec_fork_pid, SIGKILL);
    }
    if ( namespace_fork_pid > 0 ) {
        fprintf(stderr, "Singularity is sending SIGKILL to child pid: %d\n", namespace_fork_pid);

        kill(namespace_fork_pid, SIGKILL);
    }
}



int main(int argc, char ** argv) {
    FILE *loop_fp;
    FILE *containerimage_fp;
    char *containerimage;
    char *mountpoint;
    char *loop_dev;
    int retval = 0;
    uid_t uid = geteuid();

    signal(SIGINT, sighandler);
    signal(SIGKILL, sighandler);
    signal(SIGQUIT, sighandler);


    if ( uid != 0 ) {
        message(ERROR, "Calling user must be root\n");
        ABORT(1);
    }

    if ( argv[1] == NULL || argv[2] == NULL ) {
        fprintf(stderr, "USAGE: %s [singularity container image] [mount point] (shell container args)\n", argv[0]);
        return(1);
    }

    containerimage = strdup(argv[1]);
    mountpoint = strdup(argv[2]);

    if ( is_file(containerimage) < 0 ) {
        message(ERROR, "Container image not found: %s\n", containerimage);
        ABORT(1);
    }

    if ( is_dir(mountpoint) < 0 ) {
        message(ERROR, "Mount point must be a directory: %s\n", mountpoint);
        ABORT(1);
    }

    message(DEBUG, "Opening container image: %s\n", containerimage);
    if ( ( containerimage_fp = fopen(containerimage, "r+") ) < 0 ) { // Flawfinder: ignore
        message(ERROR, "Could not open image %s: %s\n", containerimage, strerror(errno));
        ABORT(255);
    }

    message(DEBUG, "Binding container to loop interface\n");
    if ( ( loop_fp = loop_bind(containerimage_fp, &loop_dev, 1)) == NULL ) {
        message(ERROR, "Could not bind image to loop!\n");
        ABORT(255);
    }

    message(DEBUG, "Forking namespace child\n");
    namespace_fork_pid = fork();
    if ( namespace_fork_pid == 0 ) {

        if ( unshare(CLONE_NEWNS) < 0 ) {
            message(ERROR, "Could not virtualize mount namespace: %s\n", strerror(errno));
            ABORT(255);
        }

        if ( mount(NULL, "/", NULL, MS_PRIVATE|MS_REC, NULL) < 0 ) {
            message(ERROR, "Could not make mountspaces private: %s\n", strerror(errno));
            ABORT(255);
        }


        if ( mount_image(loop_dev, mountpoint, 1) < 0 ) {
            message(ERROR, "Failed mounting image...\n");
            ABORT(255);
        }

        message(DEBUG, "Forking exec child\n");
        exec_fork_pid = fork();
        if ( exec_fork_pid == 0 ) {

            argv[2] = strdup("/bin/bash");

            if ( execv("/bin/bash", &argv[2]) != 0 ) { // Flawfinder: ignore (exec* is necessary)
                message(ERROR, "Exec of /bin/bash failed: %s\n", strerror(errno));
            }
            // We should never get here, so if we do, make it an error
            return(-1);

        } else if ( exec_fork_pid > 0 ) {
            int tmpstatus;

            strncpy(argv[0], "Singularity: exec", strlen(argv[0])); // Flawfinder: ignore

            message(DEBUG, "Waiting for exec child to return\n");
            waitpid(exec_fork_pid, &tmpstatus, 0);
            retval = WEXITSTATUS(tmpstatus);

            message(DEBUG, "Exec child returned (RETVAL=%d)\n", retval);

            return(retval);
        } else {
            fprintf(stderr, "ABORT: Could not exec child process: %s\n", strerror(errno));
            retval++;
        }

    } else if ( namespace_fork_pid > 0 ) {
        int tmpstatus;
        
        strncpy(argv[0], "Singularity: namespace", strlen(argv[0])); // Flawfinder: ignore
        
        message(DEBUG, "Waiting for namespace child to return\n");
        waitpid(namespace_fork_pid, &tmpstatus, 0);

        retval = WEXITSTATUS(tmpstatus);
        message(DEBUG, "Namespace child returned (RETVAL=%d)\n", retval);

    } else {
        fprintf(stderr, "ABORT: Could not fork management process: %s\n", strerror(errno));
        return(255);
    }

    return(retval);
}
