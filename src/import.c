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
#include "util/registry.h"
#include "lib/image/image.h"
#include "lib/runtime/runtime.h"
#include "util/config_parser.h"
#include "util/privilege.h"
#include "util/suid.h"
#include "util/fork.h"
#include "util/sessiondir.h"

#ifndef SYSCONFDIR
#error SYSCONFDIR not defined
#endif


int main(int argc, char **argv) {
    int retval = 0;
    char *tar_cmd[4];
    struct image_object image;
    struct image_object image_test;

    singularity_config_init(joinpath(SYSCONFDIR, "/singularity/singularity.conf"));

    singularity_priv_init();
    singularity_suid_init(argv);

    singularity_registry_init();
    singularity_priv_drop();

    if ( argc == 2 ) {
        singularity_registry_set("IMAGE", argv[1]);
    }

    image = singularity_image_init(singularity_registry_get("IMAGE"));

//    singularity_image_open(&image, O_RDWR);
//
//    singularity_image_check(&image);

    if ( image.type != EXT3 ) {
        singularity_message(ERROR, "Import is only allowed on Singularity image files\n");
        ABORT(255);
    }

    singularity_registry_set("WRITABLE", "1");

    singularity_runtime_ns(SR_NS_MNT);

//    singularity_image_bind(&image);

    if ( image.loopdev == NULL ) {
        singularity_message(ERROR, "Bind failed to connect to image!\n");
        ABORT(255);
    }

    singularity_image_mount(&image, singularity_runtime_rootfs(NULL));

    // Check to make sure the image hasn't been swapped out by a race
    image_test = singularity_image_init(singularity_registry_get("IMAGE"));
//    singularity_image_open(&image_test, O_RDONLY);
//    singularity_image_check(&image_test);
    if ( image_test.type != EXT3 ) {
        singularity_message(ERROR, "Import is only allowed on Singularity image files\n");
        ABORT(255);
    }


    if ( is_exec("/usr/bin/tar") == 0 ) {
        tar_cmd[0] = strdup("/usr/bin/tar");
    } else if ( is_exec("/bin/tar") == 0 ) {
        tar_cmd[0] = strdup("/bin/tar");
    } else {
        singularity_message(ERROR, "Could not locate the system version of 'tar'\n");
        ABORT(255);
    }

    tar_cmd[1] = strdup("-xf");
    tar_cmd[2] = strdup("-");
    tar_cmd[3] = NULL;

    if ( chdir(singularity_runtime_rootfs(NULL)) != 0 ) {
        singularity_message(ERROR, "Could not change to working directory: %s\n", singularity_runtime_rootfs(NULL));
        ABORT(255);
    }

    singularity_message(DEBUG, "Cleaning environment\n");
    if ( envclean() != 0 ) {
        singularity_message(ERROR, "Failed sanitizing the environment\n");
        ABORT(255);
    }

    singularity_priv_escalate();
    singularity_message(VERBOSE, "Opening STDIN for tar stream\n");
    retval = singularity_fork_exec(0, tar_cmd);
    singularity_priv_drop();

    if ( retval != 0 ) {
        singularity_message(ERROR, "Tar did not return successful\n");
    }

    return(retval);
}
