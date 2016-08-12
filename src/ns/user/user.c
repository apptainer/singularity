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
#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <string.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/mount.h>
#include <sys/wait.h>
#include <unistd.h>
#include <stdlib.h>
#include <sched.h>
#include <linux/limits.h>


#include "file.h"
#include "util.h"
#include "message.h"
#include "config_parser.h"
#include "privilege.h"

static int userns_enabled = 0;


int singularity_ns_user_unshare(void) {
    uid_t uid = getuid();

    if ( ( is_suid("/proc/self/exe") == 0 ) || ( getuid() == 0 ) ) {
        message(VERBOSE3, "Not virtualizing USER namespace: running privliged mode\n");
        return(0);
    }
#ifdef NS_CLONE_NEWUSER
    message(DEBUG, "Attempting to virtualize the USER namespace\n");
    if ( unshare(CLONE_NEWUSER) == 0 ) {
        message(DEBUG, "Enabling user namespaces\n");
        int child_pid = fork();

        if ( child_pid == 0 ) {

            char *map_file = (char *) malloc(PATH_MAX);
            snprintf(map_file, PATH_MAX-1, "/proc/%d/uid_map", getpid());
            FILE *uid_map = fopen(map_file, "w+");
            if ( uid_map != NULL ) {
                message(DEBUG, "Updating the child uid_map: %s\n", map_file);
                fprintf(uid_map, "%i 0 1\n", uid);
                fclose(uid_map);
                userns_enabled = 1;
            } else {
                message(ERROR, "Could not write child info to uid_map: %s\n", strerror(errno));
                ABORT(255);
            }
        } else if ( child_pid > 0 ) {
            int tmpstatus;
            int retval;

            message(DEBUG, "Waiting on USER NS child process\n");

            char *map_file = (char *) malloc(PATH_MAX);
            snprintf(map_file, PATH_MAX-1, "/proc/%d/uid_map", getpid());
            FILE *uid_map = fopen(map_file, "w+");
            if ( uid_map != NULL ) {
                message(DEBUG, "Updating the parent uid_map: %s\n", map_file);
                fprintf(uid_map, "0 %i 1\n", uid);
                fclose(uid_map);
                userns_enabled = 1;
            } else {
                message(ERROR, "Could not write parent info to uid_map: %s\n", strerror(errno));
                ABORT(255);
            }

            waitpid(child_pid, &tmpstatus, 0);

            retval = WEXITSTATUS(tmpstatus);
            exit(retval);
        } else {
            message(ERROR, "Failed to fork child for user namespace\n");
        }
    } else {
        message(VERBOSE3, "Not virtualizing USER namespace: runtime support failed\n");
    }
#else
    message(VERBOSE3, "Not virtualizing USER namespace: support not compiled in\n");
#endif

    return(0);
}


int singularity_ns_user_enabled(void) {
    return(userns_enabled);
}
