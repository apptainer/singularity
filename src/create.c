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
#include "util/fork.h"

#ifndef SYSCONFDIR
#error SYSCONFDIR not defined
#endif


int main(int argc, char **argv) {
    struct image_object image;
    long int size = 768;
    char *size_s;
    char *mkfs_cmd[7];

    singularity_config_init(joinpath(SYSCONFDIR, "/singularity/singularity.conf"));

#ifdef SUID_CREATE
    singularity_suid_init(argv);
#endif

    singularity_registry_init();
#ifdef SUID_CREATE
    singularity_priv_init();
    singularity_priv_drop();
#endif

    if ( ( size_s = singularity_registry_get("IMAGESIZE") ) != NULL ) {
        if ( str2int(size_s, &size) == 0 ) {
            singularity_message(VERBOSE, "Converted size string to long int: %ld\n", size);
        } else {
            singularity_message(ERROR, "Could not convert container size to integer\n");
            ABORT(255);
        }
    }

    singularity_message(INFO, "Initializing Singularity image subsystem\n");
    image = singularity_image_init(singularity_registry_get("IMAGE"));

    singularity_message(INFO, "Opening image file: %s\n", image.name);
    singularity_image_open(&image, O_CREAT | O_RDWR);

    singularity_message(INFO, "Creating %ldMiB image\n", size);
    singularity_image_create(&image, size);

#ifdef SUID_CREATE
    singularity_message(INFO, "Binding image to loop\n");
    singularity_image_bind(&image);
#endif

    if ( singularity_image_loopdev(&image) == NULL ) {
        singularity_message(ERROR, "Image was not bound correctly.\n");
        ABORT(255);
    }

    mkfs_cmd[0] = strdup("/sbin/mkfs.ext3");
    mkfs_cmd[1] = strdup("-q");

#ifdef SUID_CREATE
    mkfs_cmd[2] = strdup(singularity_image_loopdev(&image));
    mkfs_cmd[3] = NULL;
#else
    mkfs_cmd[2] = strdup("-E");
    // the offset in the file for the singularity header
    mkfs_cmd[3] = strjoin("offset=", int2str(strlength(LAUNCH_STRING, 1024)));
    mkfs_cmd[4] = strdup(singularity_image_path(&image));
    // pass the correct size of the file in KiB
    mkfs_cmd[5] = int2str((size*1024*1024-strlength(LAUNCH_STRING, 1024))/1024);
    mkfs_cmd[6] = NULL;
#endif

    singularity_message(DEBUG, "Cleaning environment\n");
    if ( envclean() != 0 ) {
        singularity_message(ERROR, "Failed sanitizing the environment\n");
        ABORT(255);
    }

#ifdef SUID_CREATE
    singularity_priv_escalate();
#endif
    singularity_message(INFO, "Creating file system within image\n");
    if ( singularity_fork_exec(mkfs_cmd) != 0 ) {
        singularity_message(ERROR, "Failed to create filesystem in image\n");
        ABORT(255);
    }

#ifdef SUID_CREATE
    singularity_priv_drop();
#endif

    singularity_message(INFO, "Image is done: %s\n", image.path);

    return(0);
}
