/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/
package cruntime

// #cgo CFLAGS: -I../../../core
// #cgo LDFLAGS: -L/home/mibauer/go/src/github.com/singularityware/singularity/internal/pkg/cruntime/builddir/lib -lsycore -luuid
/*
#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <errno.h>
#include <string.h>
#include <fcntl.h>

#include "builddir/config.h"
#include "lib/util/file.h"
#include "lib/util/util.h"
#include "lib/util/daemon.h"
#include "lib/util/registry.h"
#include "lib/image/image.h"
#include "lib/runtime/runtime.h"
#include "lib/util/config_parser.h"
#include "lib/util/privilege.h"
#include "lib/util/suid.h"
#include "lib/util/sessiondir.h"
#include "lib/util/cleanupd.h"

#include "action-lib/include.h"

int do_singularity(int argc, char **argv) {
    struct image_object image;
    char *pwd = get_current_dir_name();
    char *target_pwd = NULL;
    char *command = NULL;

    fd_cleanup();

    singularity_config_init();

    singularity_suid_init();
    singularity_priv_init();

    singularity_registry_init();

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

        //action_ready();

        singularity_runtime_overlayfs();
        singularity_runtime_mounts();
        singularity_runtime_files();
    } else {
        singularity_runtime_ns(SR_NS_ALL);
    }

    singularity_runtime_enter();

    singularity_runtime_environment();

    singularity_priv_drop_perm();

    if ( ( target_pwd = singularity_registry_get("TARGET_PWD") ) != NULL ) {
        singularity_message(DEBUG, "Attempting to chdir to TARGET_PWD: %s\n", target_pwd);
        if ( chdir(target_pwd) != 0 ) {
            singularity_message(ERROR, "Could not change directory to: %s\n", target_pwd);
            ABORT(255);
        }
    } else if ( singularity_registry_get("CONTAIN") != NULL ) {
        singularity_message(DEBUG, "Attempting to chdir to home: %s\n", singularity_priv_home());
        if ( chdir(singularity_priv_home()) != 0 ) {
            singularity_message(WARNING, "Could not chdir to home: %s\n", singularity_priv_home());
            if ( chdir("/") != 0 ) {
                singularity_message(ERROR, "Could not change directory within container.\n");
                ABORT(255);
            }
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
//        action_shell(argc, argv);

    // Primary Commands
    } else if ( strcmp(command, "shell") == 0 ) {
        ///action_shell(argc, argv);
    } else if ( strcmp(command, "exec") == 0 ) {
       // action_exec(argc, argv);
    } else if ( strcmp(command, "run") == 0 ) {
        //action_run(argc, argv);
    } else if ( strcmp(command, "test") == 0 ) {
        //action_test(argc, argv);
    } else {
        //singularity_message(ERROR, "Unknown action command verb was given\n");
        //ABORT(255);
    }

    return(0);
}
*/
import "C"

import ()

const (
	NS_PID = 1 << iota
	NS_IPC
	NS_MNT
	NS_UTS
	NS_USER
)

func DoSingularity(argc int, argv []string) {
	C.do_singularity(argc, argv)

}

func InitNamespaces(flags uint32) {

}

func InitMounts() {

}

func UnshareNamespaces(flags uint32) {

}

func OverlayFS() {

}
