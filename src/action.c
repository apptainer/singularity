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


#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <errno.h>
#include <string.h>
#include <fcntl.h>

#include "config.h"
#include "util/file.h"
#include "util/util.h"
#include "util/daemon.h"
#include "util/registry.h"
#include "lib/image/image.h"
#include "lib/runtime/runtime.h"
#include "util/config_parser.h"
#include "util/privilege.h"
#include "util/suid.h"
#include "util/sessiondir.h"
#include "util/cleanupd.h"

#include "./action-lib/include.h"

#ifndef SYSCONFDIR
#error SYSCONFDIR not defined
#endif


int main(int argc, char **argv) {
    struct image_object image;
    char *pwd = get_current_dir_name();
    char *target_pwd = NULL;
    char *command = NULL;

    singularity_config_init(joinpath(SYSCONFDIR, "/singularity/singularity.conf"));

    singularity_priv_init();
    singularity_suid_init(argv);

    singularity_registry_init();
    
    singularity_priv_userns();
    singularity_priv_drop();

    singularity_runtime_autofs();

    singularity_daemon_init();

    if ( singularity_registry_get("WRITABLE") != NULL ) {
        singularity_message(VERBOSE3, "Instantiating writable container image object\n");
        image = singularity_image_init(singularity_registry_get("IMAGE"), O_RDWR);
    } else {
        singularity_message(VERBOSE3, "Instantiating read only container image object\n");
        image = singularity_image_init(singularity_registry_get("IMAGE"), O_RDONLY);
    }

    if ( singularity_registry_get("DAEMON_JOIN") == NULL ) {
        singularity_cleanupd();

        singularity_runtime_ns(SR_NS_ALL);

        singularity_sessiondir();

        singularity_image_mount(&image, CONTAINER_MOUNTDIR);

        action_ready();

        singularity_runtime_overlayfs();
        singularity_runtime_mounts();
        singularity_runtime_files();
    } else {
        singularity_runtime_ns(SR_NS_ALL);
    }

    singularity_runtime_enter();
    
    singularity_runtime_environment();
    
    singularity_priv_drop_perm();

    if ( singularity_registry_get("CONTAIN") != NULL ) {
        singularity_message(DEBUG, "Attempting to chdir to home: %s\n", singularity_priv_home());
        if ( chdir(singularity_priv_home()) != 0 ) {
            singularity_message(WARNING, "Could not chdir to home: %s\n", singularity_priv_home());
            if ( chdir("/") != 0 ) {
                singularity_message(ERROR, "Could not change directory within container.\n");
                ABORT(255);
            }
        }
    } else if ( ( target_pwd = singularity_registry_get("TARGET_PWD") ) != NULL ) {
        singularity_message(DEBUG, "Attempting to chdir to TARGET_PWD: %s\n", target_pwd);
        if ( chdir(target_pwd) != 0 ) {
            singularity_message(ERROR, "Could not change directory to: %s\n", target_pwd);
            ABORT(255);
        }
    } else if ( pwd != NULL ) {
        singularity_message(DEBUG, "Attempting to chdir to CWD: %s\n", pwd);
        if ( chdir(pwd) != 0 ) {
            singularity_message(VERBOSE, "Could not chdir to current dir: %s\n", pwd);
            if ( chdir(singularity_priv_home()) != 0 ) {
                singularity_message(WARNING, "Could not chdir to home: %s\n", singularity_priv_home());
                if ( chdir("/") != 0 ) {
                    singularity_message(ERROR, "Could not change directory within container.\n");
                    ABORT(255);
                }
            }
        }
    } else {
        singularity_message(ERROR, "Could not obtain current directory.\n");
        ABORT(255);
    }

    free(target_pwd);

    command = singularity_registry_get("COMMAND");

    envar_set("SINGULARITY_CONTAINER", singularity_image_name(&image), 1); // Legacy PS1 support
    envar_set("SINGULARITY_NAME", singularity_image_name(&image), 1);
    envar_set("SINGULARITY_SHELL", singularity_registry_get("SHELL"), 1);
    envar_set("SINGULARITY_APPNAME", singularity_registry_get("APPNAME"), 1);

    singularity_message(LOG, "USER=%s, IMAGE='%s', COMMAND='%s'\n", singularity_priv_getuser(), singularity_image_name(&image), singularity_registry_get("COMMAND"));

    if ( command == NULL ) {
        singularity_message(INFO, "No action command verb was given, invoking 'shell'\n");
        action_shell(argc, argv);

    // Primary Commands
    } else if ( strcmp(command, "shell") == 0 ) {
        action_shell(argc, argv);
    } else if ( strcmp(command, "exec") == 0 ) {
        action_exec(argc, argv);
    } else if ( strcmp(command, "inspect") == 0 ) {
        action_exec(argc, argv);
    } else if ( strcmp(command, "run") == 0 ) {
        action_run(argc, argv);
    } else if ( strcmp(command, "test") == 0 ) {
        action_test(argc, argv);
    } else {
        singularity_message(ERROR, "Unknown action command verb was given\n");
        ABORT(255);
    }

    return(0);
}
