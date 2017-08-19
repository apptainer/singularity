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


void action_apprun(int argc, char **argv) {
    singularity_message(VERBOSE, "Starting app runscript\n");

    if ( is_exec("/.singularity.d/actions/apprun") == 0 ) {
        singularity_message(DEBUG, "Exec'ing /.singularity.d/actions/apprun\n");
        if ( execv("/.singularity.d/actions/apprun", argv) < 0 ) { // Flawfinder: ignore
            singularity_message(ERROR, "There is no apprun driver inside the container.: %s\n", strerror(errno));
            ABORT(255);
        }
    } else {
        singularity_message(ERROR, "No apprun driver found inside container\n");
        ABORT(255);
    }

    singularity_message(ERROR, "Oh dear, should not have gotten here.\n");
    ABORT(255);
}
