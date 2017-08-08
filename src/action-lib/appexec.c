/* 
 * Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
 * Copyright (c) 2017, Vanessa Sochat. All rights reserved
 *
 *
 * See the COPYRIGHT.md file at the top-level directory of this distribution and at
 * https://github.com/singularityware/singularity/blob/master/COPYRIGHT.md.
 *
 * This file is part of the Singularity Linux container project. It is subject to the license
 * terms in the LICENSE.md file found in the top-level directory of this distribution and
 * at https://github.com/singularityware/singularity/blob/master/LICENSE.md. No part
 * of Singularity, including this file, may be copied, modified, propagated, or distributed
 * except according to the terms contained in the LICENSE.md file.
 *
 *
*/

#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <string.h>
#include <sys/stat.h>
#include <unistd.h>
#include <stdlib.h>

#include "util/file.h"
#include "util/util.h"
#include "util/message.h"


void action_appexec(int argc, char **argv) {

    if ( argc <= 1 ) {
        singularity_message(ERROR, "No program name to exec\n");
        ABORT(255);
    }

    singularity_message(DEBUG, "Checking for: /.singularity.d/actions/appexec\n");
    if ( is_exec("/.singularity.d/actions/appexec") == 0 ) {
        singularity_message(VERBOSE, "Exec'ing /.singularity.d/actions/appexec\n");
        if ( execv("/.singularity.d/actions/appexec", argv) < 0 ) { // Flawfinder: ignore
            singularity_message(ERROR, "Failed to execv() /.singularity.d/actions/appexec: %s\n", strerror(errno));
        }
    } else {
        singularity_message(ERROR, "No appexec driver found inside container\n");
        ABORT(255);
    }

    singularity_message(ERROR, "Oh dear, should not have gotten here.\n");
    ABORT(255);

}
