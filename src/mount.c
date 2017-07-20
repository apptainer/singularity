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

    image = singularity_image_init(singularity_registry_get("IMAGE"));

    if ( singularity_registry_get("WRITABLE") == NULL ) {
        singularity_image_open(&image, O_RDONLY);
    } else {
        singularity_image_open(&image, O_RDWR);
    }

    singularity_image_check(&image);

    if ( image.type != SINGULARITY ) {
        singularity_message(ERROR, "Mount is only allowed on Singularity image files\n");
        ABORT(255);
    }

    if ( is_owner(singularity_runtime_rootfs(NULL), 0) != 0 ) {
        singularity_message(ERROR, "Root must own container mount directory: %s\n", singularity_runtime_rootfs(NULL));
        ABORT(255);
    }

    if ( argc > 1 ) {
        singularity_runtime_ns(SR_NS_MNT);

        singularity_image_bind(&image);
        singularity_image_mount(&image, singularity_runtime_rootfs(NULL));

        singularity_priv_drop_perm();

        envar_set("SINGULARITY_MOUNTPOINT", singularity_runtime_rootfs(NULL), 1);

        singularity_message(VERBOSE, "Running command: %s\n", argv[1]);
        singularity_message(DEBUG, "Calling exec...\n");
        execvp(argv[1], &argv[1]); // Flawfinder: ignore (Yes flawfinder, we are exec'ing)

        singularity_message(ERROR, "Exec failed: %s: %s\n", argv[1], strerror(errno));
        ABORT(255);
    } else {
        singularity_runtime_ns(SR_NS_MNT);

        singularity_image_bind(&image);
        singularity_image_mount(&image, singularity_runtime_rootfs(NULL));

        singularity_priv_escalate();
        if ( mount(singularity_runtime_rootfs(NULL), singularity_runtime_rootfs(NULL), NULL, MS_BIND|MS_NOSUID|MS_NODEV|MS_REC, NULL) < 0 ) {
            singularity_message(ERROR, "There was an error binding mounted container to %s: %s\n", singularity_runtime_rootfs(NULL), strerror(errno));
            ABORT(255);
        }

        singularity_priv_drop_perm();

        singularity_message(INFO, "%s is mounted at: %s\n\n", singularity_image_name(&image), singularity_runtime_rootfs(NULL));
        envar_set("PS1", "Singularity> ", 1);

        execl("/bin/sh", "/bin/sh", NULL); // Flawfinder: ignore (Yes flawfinder, this is what we want, sheesh, so demanding!)

        singularity_message(ERROR, "Exec of /bin/sh failed: %s\n", strerror(errno));
        ABORT(255);
    }

    return(0);
}
