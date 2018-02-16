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
#include <sys/mount.h>

#include "config.h"
#include "util/file.h"
#include "util/util.h"
#include "util/registry.h"
#include "util/config_parser.h"
#include "util/privilege.h"
#include "util/suid.h"
#include "lib/image/image.h"
#include "lib/runtime/runtime.h"

#ifndef SYSCONFDIR
#error SYSCONFDIR not defined
#endif


int main(int argc, char **argv) {
    struct image_object image;

    singularity_config_init(joinpath(SYSCONFDIR, "/singularity/singularity.conf"));

    singularity_priv_init();
    singularity_suid_init(argv);

    singularity_registry_init();
    singularity_priv_drop();

    singularity_runtime_autofs();

    if ( singularity_registry_get("WRITABLE") != NULL ) {
        singularity_message(VERBOSE3, "Instantiating writable container image object\n");
        image = singularity_image_init(singularity_registry_get("IMAGE"), O_RDWR);
    } else {
        singularity_message(VERBOSE3, "Instantiating read only container image object\n");
        image = singularity_image_init(singularity_registry_get("IMAGE"), O_RDONLY);
    }

    if ( is_owner(CONTAINER_MOUNTDIR, 0) != 0 ) {
        singularity_message(ERROR, "Root must own container mount directory: %s\n", CONTAINER_MOUNTDIR);
        ABORT(255);
    }

    singularity_runtime_ns(SR_NS_MNT);

    singularity_image_mount(&image, CONTAINER_MOUNTDIR);

    singularity_runtime_overlayfs();

    singularity_priv_drop_perm();

    envar_set("SINGULARITY_MOUNTPOINT", CONTAINER_FINALDIR, 1);

    if ( argc > 1 ) {

        singularity_message(VERBOSE, "Running command: %s\n", argv[1]);
        singularity_message(DEBUG, "Calling exec...\n");
        execvp(argv[1], &argv[1]); // Flawfinder: ignore (Yes flawfinder, we are exec'ing)

        singularity_message(ERROR, "Exec failed: %s: %s\n", argv[1], strerror(errno));
        ABORT(255);

    } else {

        singularity_message(INFO, "%s is mounted at: %s\n\n", singularity_image_name(&image), CONTAINER_FINALDIR);
        envar_set("PS1", "Singularity> ", 1);

        execl("/bin/sh", "/bin/sh", NULL); // Flawfinder: ignore (Yes flawfinder, this is what we want, sheesh, so demanding!)

        singularity_message(ERROR, "Exec of /bin/sh failed: %s\n", strerror(errno));
        ABORT(255);
    }

    return(0);
}
