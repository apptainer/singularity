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
#include "util/sessiondir.h"

#include "./bootstrap-lib/include.h"

#ifndef SYSCONFDIR
#error SYSCONFDIR not defined
#endif


int main(int argc, char **argv) {
    struct image_object image;

    singularity_config_init(joinpath(SYSCONFDIR, "/singularity/singularity.conf"));
    singularity_registry_init();
    singularity_priv_init();

    singularity_message(INFO, "Sanitizing environment\n");
    if ( envclean() != 0 ) {
        singularity_message(ERROR, "Failed sanitizing the environment\n");
        ABORT(255);
    }

    singularity_registry_set("WRITABLE", "1");

    image = singularity_image_init(singularity_registry_get("IMAGE"));

//    singularity_image_open(&image, O_RDWR);
//
//    singularity_image_check(&image);

    singularity_runtime_ns(SR_NS_MNT);

//    singularity_image_bind(&image);
    singularity_image_mount(&image, CONTAINER_MOUNTDIR);

    envar_set("PATH", "/bin:/sbin:/usr/bin:/usr/sbin:/usr/local/bin:/usr/local/sbin", 1);
    envar_set("SINGULARITY_ROOTFS", CONTAINER_MOUNTDIR, 1);
    envar_set("SINGULARITY_libexecdir", singularity_registry_get("LIBEXECDIR"), 1);
    envar_set("SINGULARITY_IMAGE", singularity_registry_get("IMAGE"), 1);
    envar_set("SINGULARITY_BUILDDEF", singularity_registry_get("BUILDDEF"), 1);
    envar_set("SINGULARITY_CHECKS", singularity_registry_get("CHECKS"), 1);
    envar_set("SINGULARITY_CHECKLEVEL", singularity_registry_get("CHECKLEVEL"), 1);
    envar_set("SINGULARITY_CHECKTAGS", singularity_registry_get("CHECKTAGS"), 1);
    envar_set("SINGULARITY_MESSAGELEVEL", singularity_registry_get("MESSAGELEVEL"), 1);
    envar_set("SINGULARITY_NOTEST", singularity_registry_get("NOTEST"), 1);
    envar_set("SINGULARITY_BUILDSECTION", singularity_registry_get("BUILDSECTION"), 1);
    envar_set("SINGULARITY_BUILDNOBASE", singularity_registry_get("BUILDNOBASE"), 1);
    envar_set("SINGULARITY_DOCKER_PASSWORD", singularity_registry_get("DOCKER_PASSWORD"), 1);
    envar_set("SINGULARITY_DOCKER_USERNAME", singularity_registry_get("DOCKER_USERNAME"), 1);
    envar_set("SINGULARITY_CACHEDIR", singularity_registry_get("CACHEDIR"), 1);
    envar_set("SINGULARITY_version", singularity_registry_get("VERSION"), 1);
    envar_set("HOME", singularity_priv_home(), 1);
    envar_set("LANG", "C", 1);

    // At this point, the container image is mounted at
    // CONTAINER_MOUNTDIR, and bootstrap code can be added
    // in the bootstrap-lib/ directory.

    bootstrap_init(argc, argv);

    return(0);
}
