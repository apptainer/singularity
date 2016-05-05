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
#include "loop-control.h"


#ifndef LIBEXECDIR
#define LIBEXECDIR "undefined"
#endif
#ifndef SYSCONFDIR
#define SYSCONFDIR "/etc"
#endif
#ifndef LOCALSTATEDIR
#define LOCALSTATEDIR "/var/"
#endif


// Yes, I know... Global variables suck but necessary to pass sig to child
pid_t child_pid = 0;


void sighandler(int sig) {
    signal(sig, sighandler);

    printf("Caught signal: %d\n", sig);
    fflush(stdout);

    if ( child_pid > 0 ) {
        printf("Singularity is sending SIGKILL to child pid: %d\n", child_pid);
        fflush(stdout);

        kill(child_pid, SIGKILL);
    }
}


int main(int argc, char ** argv) {
    char *containerimage;
    char *mountpoint;
    char *loop_dev;
    int retval = 0;
    int containerimage_fd;
    uid_t uid = geteuid();

    if ( uid != 0 ) {
        fprintf(stderr, "ABORT: Calling user must be root\n");
        return(1);
    }

    if ( argv[1] == NULL || argv[2] == NULL ) {
        fprintf(stderr, "USAGE: %s [singularity container image] [mount point]\n", argv[0]);
        return(1);
    }

    containerimage = strdup(argv[1]);
    mountpoint = strdup(argv[2]);

    if ( is_file(containerimage) < 0 ) {
        fprintf(stderr, "ABORT: Container image not found: %s\n", containerimage);
        return(1);
    }

    if ( is_dir(mountpoint) < 0 ) {
        fprintf(stderr, "ABORT: Mount point must be a directory: %s\n", mountpoint);
        return(1);
    }

    if ( unshare(CLONE_NEWNS) < 0 ) {
        fprintf(stderr, "ABORT: Could not virtulize mount namespace\n");
        return(255);
    }

    if ( mount(NULL, "/", NULL, MS_PRIVATE|MS_REC, NULL) < 0 ) {
        // I am not sure if this error needs to be caught, maybe it will fail
        // on older kernels? If so, we can fix then.
        fprintf(stderr, "ABORT: Could not make mountspaces private: %s\n", strerror(errno));
        return(255);
    }


    if ( ( containerimage_fd = open(containerimage, O_RDWR) ) < 0 ) {
        fprintf(stderr, "ERROR: Could not open image %s: %s\n", containerimage, strerror(errno));
        return(255);
    }

    loop_dev = obtain_loop_dev();
    if ( associate_loop(containerimage_fd, loop_dev) < 0 ) {
        fprintf(stderr, "ERROR: Could not associate %s to loop device %s\n", containerimage, loop_dev);
        return(255);
    }

    if ( mount_image(loop_dev, mountpoint, 1) < 0 ) {
        fprintf(stderr, "ABORT: exiting...\n");
        return(255);
    }

    child_pid = fork();

    if ( child_pid == 0 ) {
        char *prompt;
        char *containername = basename(strdup(containerimage));
        
        prompt = (char *) malloc(strlen(containerimage) + strlen(mountpoint) + 15);
        snprintf(prompt, strlen(containerimage) + strlen(mountpoint) + 15, "Singularity/%s> ", containername, mountpoint);

        setenv("PS1", prompt, 1);

        if ( getenv("SINGULARITY_QUIET") == NULL ) {
            fprintf(stderr, "\nMounting image %s at %s\n\n", containerimage, mountpoint);
            fprintf(stderr, "Invoking a new shell to which the mountpoint will be visible. When\n");
            fprintf(stderr, "this shell exits, the mount point will automatically be unmounted.\n\n");
        }

        argv[2] = strdup("/bin/bash");

        if ( execv("/bin/bash", &argv[2]) != 0 ) {
            fprintf(stderr, "ABORT: exec of /bin/bash failed: %s\n", strerror(errno));
        }

    } else if ( child_pid > 0 ) {
        int tmpstatus;
        signal(SIGINT, sighandler);
        signal(SIGKILL, sighandler);
        signal(SIGQUIT, sighandler);

        waitpid(child_pid, &tmpstatus, 0);
        retval = WEXITSTATUS(tmpstatus);

    } else {
        fprintf(stderr, "ABORT: Could not fork child process\n");
        retval++;
    }

    return(retval);
}
