/*
 * Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
 *
 * See the COPYRIGHT.md file at the top-level directory of this distribution and at
 * https://github.com/singularityware/singularity/blob/master/COPYRIGHT.md.
 *
 * This file is part of the Singularity Linux container project. It is subject to the license
 * terms in the LICENSE.md file found in the top-level directory of this distribution and
 * at https://github.com/singularityware/singularity/blob/master/LICENSE.md. No part
 * of Singularity, including this file, may be copied, modified, propagated, or distributed
 * except according to the terms contained in the LICENSE.md file.
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
    int ret;
    struct image_object image;

    singularity_config_init(joinpath(SYSCONFDIR, "/singularity/singularity.conf"));

    singularity_priv_init();
    singularity_suid_init(argv);

    singularity_registry_init();
    singularity_priv_drop();

    singularity_message(INFO, "Initializing Singularity image subsystem\n");
    image = singularity_image_init(singularity_registry_get("IMAGE"));

    singularity_message(INFO, "Opening image file: %s\n", image.name);
    singularity_image_open(&image, O_RDWR);

    if ( image.vbpresent == 1 ) {
        ret = singularity_image_verify(&image);
    } else {
        singularity_message(ERROR, "The image was not created with a verification block needed by the signature feature\n");
        ABORT(255);
    }

    if (ret < 0) {
        singularity_message(ERROR, "Could not authenticate/validate image\n");
    } else {
        singularity_message(INFO, "Signature and checksum validated, all good\n");
    }

    return(0);
}
