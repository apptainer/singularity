/* 
 * Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
 * 
 * This software is licensed under a 3-clause BSD license.  Please
 * consult LICENSE file distributed with the sources of this project regarding
 * your rights to use or distribute this software.
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
#include "util/fork.h"
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

    char **exec_arg = malloc(sizeof(char *) * 3);
    exec_arg[0] = joinpath(LIBEXECDIR, "/singularity/bin/sinit"); //path to sinit binary
    exec_arg[2] = '\0';

    singularity_config_init(joinpath(SYSCONFDIR, "/singularity/singularity.conf"));

    singularity_priv_init();
    singularity_suid_init(argv);

    singularity_registry_init();
    singularity_priv_userns();
    singularity_priv_drop();

    singularity_registry_set("UNSHARE_PID", "1");
    singularity_registry_set("UNSHARE_IPC", "1");
        
    singularity_runtime_ns(SR_NS_ALL);
    
    singularity_sessiondir();
    singularity_cleanupd();
    
    image = singularity_image_init(singularity_registry_get("IMAGE"));

    if ( singularity_registry_get("WRITABLE") == NULL ) {
        singularity_image_open(&image, O_RDONLY);
    } else {
        singularity_image_open(&image, O_RDWR);
    }

    singularity_image_check(&image);
    singularity_image_bind(&image);
    singularity_image_mount(&image, singularity_runtime_rootfs(NULL));

    action_ready(singularity_runtime_rootfs(NULL));

    singularity_runtime_overlayfs();
    singularity_runtime_mounts();
    singularity_runtime_files();

    exec_arg[1] = strdup(singularity_runtime_rootfs(NULL));
    
    if ( execv(exec_arg[0], exec_arg) < 0 ) { //Flawfinder: ignore
        singularity_message(ERROR, "Failed to exec sinit: %s\n", strerror(errno));
        ABORT(255);
    }
    
    return(0);
}
