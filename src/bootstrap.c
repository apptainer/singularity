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
#include "util/sessiondir.h"

#include "./bootstrap-lib/include.h"

#ifndef SYSCONFDIR
#error SYSCONFDIR not defined
#endif


int main(int argc, char **argv) {
    struct image_object image;
    char *lang = envar_get("LANG", "_-=+:,.%", 128);
    char *term = envar_get("TERM", "-", 128);

    singularity_config_init(joinpath(SYSCONFDIR, "/singularity/singularity.conf"));
    singularity_registry_init();
    singularity_priv_init();

    singularity_message(INFO, "Sanitizing environment\n");
    if ( envclean() != 0 ) {
        singularity_message(ERROR, "Failed sanitizing the environment\n");
        ABORT(255);
    }

    envar_set("PATH", "/bin:/sbin:/usr/bin:/usr/sbin:/usr/local/bin:/usr/local/sbin", 1);
    envar_set("SINGULARITY_libexecdir", singularity_registry_get("LIBEXECDIR"), 1);
    envar_set("SINGULARITY_IMAGE", singularity_registry_get("IMAGE"), 1);
    envar_set("SINGULARITY_BUILDDEF", singularity_registry_get("BUILDDEF"), 1);
    envar_set("SINGULARITY_MESSAGELEVEL", singularity_registry_get("MESSAGELEVEL"), 1);
    envar_set("SINGULARITY_version", singularity_registry_get("VERSION"), 1);
    envar_set("LANG", lang, 1);
    envar_set("TERM", term, 1);

    singularity_message(INFO, "Setting envar: 'HOME' = '%s'\n", singularity_priv_home());
    envar_set("HOME", singularity_priv_home(), 1);

//    singularity_registry_set("WRITABLE", "1");

//    singularity_sessiondir();

    image = singularity_image_init(singularity_registry_get("IMAGE"));

    singularity_image_open(&image, O_RDWR);

    singularity_runtime_ns(SR_NS_MNT);

    singularity_image_bind(&image);
    singularity_image_mount(&image, singularity_runtime_rootfs(NULL));

    singularity_message(DEBUG, "Setting SINGULARITY_ROOTFS to: %s\n", singularity_runtime_rootfs(NULL));
    envar_set("SINGULARITY_ROOTFS", singularity_runtime_rootfs(NULL), 1);

    // At this point, the container image is mounted at
    // singularity_runtime_rootfs(NULL), and bootstrap code can be added
    // in the bootstrap-lib/ directory.

    bootstrap_init(argc, argv);

    return(0);
}
