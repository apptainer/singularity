/* 
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
#include "util/registry.h"
#include "lib/image/image.h"
#include "lib/runtime/runtime.h"
#include "util/config_parser.h"
#include "util/privilege.h"
#include "util/suid.h"

#include "./action-lib/include.h"

#ifndef SYSCONFDIR
#error SYSCONFDIR not defined
#endif


int main(int argc, char **argv) {
    struct image_object image;
    char *command;
    char *dir = get_current_dir_name();

    singularity_suid_init();

    singularity_config_init(joinpath(SYSCONFDIR, "/singularity/singularity.conf"));
    singularity_registry_init();
    singularity_priv_init();
    singularity_priv_drop();

    image = singularity_image_init(singularity_registry_get("CONTAINER"));

    singularity_runtime_tmpdir(singularity_image_sessiondir(&image));
    singularity_runtime_ns();

    if ( singularity_registry_get("WRITABLE") == NULL ) {
        singularity_image_open(&image, O_RDONLY);
    } else {
        singularity_image_open(&image, O_RDWR);
    }

    singularity_image_bind(&image);
    singularity_image_mount(&image, singularity_runtime_containerdir(NULL));

    action_ready(singularity_runtime_containerdir(NULL));

    singularity_runtime_overlayfs();
    singularity_runtime_mounts();
    singularity_runtime_files();
    singularity_runtime_enter();

    singularity_runtime_environment();

    singularity_priv_drop_perm();

    if ( is_dir(dir) == 0 ) {
        chdir(dir);
    } else {
        singularity_message(VERBOSE, "Current directory is not available within container, landing in home\n");
        chdir(singularity_priv_home());
    }

    setenv("HISTFILE", "/dev/null", 1);
    setenv("SINGULARITY_CONTAINER", singularity_image_name(&image), 1);
    command = singularity_registry_get("COMMAND");
    
    if ( command == NULL ) {
        singularity_message(INFO, "No action command verb was given, invoking 'shell'\n");
        action_shell(argc, argv);
    } else if ( strcmp(command, "shell") == 0 ) {
        action_shell(argc, argv);
    } else if ( strcmp(command, "exec") == 0 ) {
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
