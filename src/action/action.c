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
#include <linux/limits.h>
#include <unistd.h>
#include <stdlib.h>

#include "file.h"
#include "util.h"
#include "message.h"
#include "privilege.h"
#include "exec/exec.h"
#include "shell/shell.h"
#include "run/run.h"

#define ACTION_SHELL    1
#define ACTION_EXEC     2
#define ACTION_RUN      3

static int action = 0;
static char *cwd_path;

int singularity_action_init(void) {
    char *command = getenv("SINGULARITY_COMMAND");
    message(DEBUG, "Checking on action to run\n");

    unsetenv("SINGULARITY_COMMAND");

    if ( command == NULL ) {
        message(ERROR, "SINGULARITY_COMMAND is undefined\n");
        ABORT(1);
    } else if ( strcmp(command, "shell") == 0 ) {
        message(DEBUG, "Setting action to: shell\n");
        action = ACTION_SHELL;
    } else if ( strcmp(command, "exec") == 0 ) {
        message(DEBUG, "Setting action to: exec\n");
        action = ACTION_EXEC;
    } else if ( strcmp(command, "run") == 0 ) {
        message(DEBUG, "Setting action to: run\n");
        action = ACTION_RUN;
    } else {
        message(ERROR, "Unknown container action: %s\n", command);
        ABORT(1);
    }

    cwd_path = (char *) malloc(sizeof(char) * PATH_MAX);

//    message(DEBUG, "Obtaining file descriptor to current directory\n");
//    if ( (cwd_fd = open(".", O_RDONLY)) < 0 ) { // Flawfinder: ignore (need current directory FD)
//        message(ERROR, "Could not open cwd fd (%s)!\n", strerror(errno));
//        ABORT(1);
//    }
    message(DEBUG, "Getting current working directory path string\n");
    if ( getcwd(cwd_path, PATH_MAX) == NULL ) {
        message(ERROR, "Could not obtain current directory path: %s\n", strerror(errno));
        ABORT(1);
    }

    return(0);
}

int singularity_action_do(int argc, char **argv) {

    priv_drop_perm();

    if ( chdir(cwd_path) < 0 ) {
        message(WARNING, "Could not chdir to: %s\n", cwd_path);
    }

    if ( action == ACTION_SHELL ) {
        message(DEBUG, "Running action: shell\n");
        action_shell_do(argc, argv);
    } else if ( action == ACTION_EXEC ) {
        message(DEBUG, "Running action: exec\n");
        action_exec_do(argc, argv);
    } else if ( action == ACTION_RUN ) {
        message(DEBUG, "Running action: run\n");
        action_run_do(argc, argv);
    }
    message(ERROR, "Called singularity_action_do() without singularity_action_init()\n");
    return(-1);
}
