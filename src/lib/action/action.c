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

#include "util/file.h"
#include "util/util.h"
#include "lib/message.h"
#include "lib/privilege.h"
#include "exec/exec.h"
#include "shell/shell.h"
#include "run/run.h"
#include "start/start.h"
#include "stop/stop.h"
#include "test/test.h"

#define ACTION_SHELL    1
#define ACTION_EXEC     2
#define ACTION_RUN      3
#define ACTION_TEST     4
#define ACTION_START    5
#define ACTION_STOP     6

static int action = 0;
static char *cwd_path;

static char *LD_UNSECVARS[] = {
    "GCONV_PATH",
    "GETCONF_DIR",
    "HOSTALIASES",
    "LD_AUDIT",
    "LD_DEBUG",
    "LD_DEBUG_OUTPUT",
    "LD_DYNAMIC_WEAK",
    "LD_LIBRARY_PATH",
    "LD_ORIGIN_PATH",
    "LD_PRELOAD",
    "LD_PROFILE",
    "LD_SHOW_AUXV",
    "LD_USE_LOAD_BIAS",
    "LOCALDOMAIN",
    "LOCPATH",
    "MALLOC_TRACE",
    "NIS_PATH",
    "NLSPATH",
    "RESOLV_HOST_CONF",
    "RES_OPTIONS",
    "TMPDIR",
    "TZDIR",
};

static int LD_UNSECVARS_LEN = 22;
static int LD_UNSECVARS_MAX_STRLEN = 16;
static int LD_UNSECVARS_PREFIX_LEN = 10;
static char * LD_UNSECVARS_PREFIX = "CONTAINER_%s";

int singularity_proxy_unsecvars() {
    int prefixedvar_max_len = LD_UNSECVARS_MAX_STRLEN + LD_UNSECVARS_PREFIX_LEN + 1;
    char prefixedvar[prefixedvar_max_len];
    char *varname;
    char *value;
    int c;
    for (c = 0; c < LD_UNSECVARS_LEN; c++) {
        varname = LD_UNSECVARS[c];
        snprintf(prefixedvar, prefixedvar_max_len, LD_UNSECVARS_PREFIX, varname);
        value = getenv(prefixedvar);
        if ( value != NULL ) {
             singularity_message(VERBOSE, "Setting %s to `%s`\n", varname, value);
             setenv(varname, value, 1);
        }
        unsetenv(prefixedvar);
    }
    return 0;
}

int singularity_action_init(void) {
    singularity_proxy_unsecvars();

    char *command = envar("SINGULARITY_COMMAND", "", 10);
    singularity_message(DEBUG, "Checking on action to run\n");

    unsetenv("SINGULARITY_COMMAND");

    if ( command == NULL ) {
        singularity_message(ERROR, "SINGULARITY_COMMAND is undefined\n");
        ABORT(1);
    } else if ( strcmp(command, "shell") == 0 ) {
        singularity_message(DEBUG, "Setting action to: shell\n");
        action = ACTION_SHELL;
        action_shell_init();
    } else if ( strcmp(command, "exec") == 0 ) {
        singularity_message(DEBUG, "Setting action to: exec\n");
        action = ACTION_EXEC;
        action_exec_init();
    } else if ( strcmp(command, "run") == 0 ) {
        singularity_message(DEBUG, "Setting action to: run\n");
        action = ACTION_RUN;
        action_run_init();
    } else if ( strcmp(command, "test") == 0 ) {
        singularity_message(DEBUG, "Setting action to: test\n");
        action = ACTION_TEST;
        action_test_init();
    } else if ( strcmp(command, "start") == 0 ) {
        singularity_message(DEBUG, "Setting action to: start\n");
        action = ACTION_START;
        action_start_init();
    } else if ( strcmp(command, "stop") == 0 ) {
        singularity_message(DEBUG, "Setting action to: stop\n");
        action = ACTION_STOP;
        action_stop_init();
    } else {
        singularity_message(ERROR, "Unknown container action: %s\n", command);
        ABORT(1);
    }

    free(command);

    cwd_path = (char *) malloc(PATH_MAX);

    singularity_message(DEBUG, "Getting current working directory path string\n");
    if ( getcwd(cwd_path, PATH_MAX) == NULL ) {
        singularity_message(ERROR, "Could not obtain current directory path: %s\n", strerror(errno));
        ABORT(1);
    }

    return(0);
}

int singularity_action_do(int argc, char **argv) {

    singularity_priv_drop_perm();

    singularity_message(DEBUG, "Trying to change directory to where we started\n");
    char *target_pwd = envar_path("SINGULARITY_TARGET_PWD");
    if (!target_pwd || (chdir(target_pwd) < 0)) {
        if ( chdir(cwd_path) < 0 ) {
            char *homedir;

            singularity_message(DEBUG, "Failed changing directory to: %s\n", cwd_path);
            singularity_message(VERBOSE2, "Changing to home directory\n");

            singularity_message(DEBUG, "Obtaining user's homedir\n");
            homedir = get_homedir(NULL);

            if ( ( homedir != NULL ) && ( chdir(homedir) < 0 ) ) {
                singularity_message(WARNING, "Could not chdir to home directory: %s\n", homedir);
            }
        }
    }
    free(target_pwd);

    if ( action == ACTION_SHELL ) {
        singularity_message(DEBUG, "Running action: shell\n");
        action_shell_do(argc, argv);
    } else if ( action == ACTION_EXEC ) {
        singularity_message(DEBUG, "Running action: exec\n");
        action_exec_do(argc, argv);
    } else if ( action == ACTION_RUN ) {
        singularity_message(DEBUG, "Running action: run\n");
        action_run_do(argc, argv);
    } else if ( action == ACTION_TEST ) {
        singularity_message(DEBUG, "Running action: test\n");
        action_test_do(argc, argv);
    } else if ( action == ACTION_START ) {
        singularity_message(DEBUG, "Running action: start\n");
        action_start_do(argc, argv);
    } else if ( action == ACTION_STOP ) {
        singularity_message(DEBUG, "Running action: stop\n");
        action_stop_do(argc, argv);
    }
    singularity_message(ERROR, "Called singularity_action_do() without singularity_action_init()\n");
    return(-1);
}
