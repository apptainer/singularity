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


void action_appshell(int argc, char **argv) {

    singularity_message(INFO, "Singularity: Invoking an interactive shell into application...\n\n");

    if ( is_exec("/.singularity.d/actions/shell") == 0 ) {
        singularity_message(DEBUG, "Exec'ing /.singularity.d/actions/shell\n");
        if ( execv("/.singularity.d/actions/shell", argv) < 0 ) { // Flawfinder: ignore
            singularity_message(ERROR, "Failed to execv() /.singularity.d/actions/shell, continuing to /bin/sh: %s\n", strerror(errno));
        }
    } else if ( is_exec("/.shell") == 0 ) {
        singularity_message(DEBUG, "Exec'ing /.shell\n");
        if ( execv("/.shell", argv) < 0 ) { // Flawfinder: ignore
            singularity_message(ERROR, "Failed to execv() /.shell, continuing to /bin/sh: %s\n", strerror(errno));
        }
    }

    singularity_message(VERBOSE, "Invoking the container's /bin/sh\n");
    if ( is_exec("/bin/sh") == 0 ) {
        singularity_message(DEBUG, "Exec'ing /bin/sh\n");
        argv[0] = strdup("/bin/sh");
        if ( execv("/bin/sh", argv) < 0 ) { // Flawfinder: ignore
            singularity_message(ERROR, "Failed to execv() /bin/sh: %s\n", strerror(errno));
            ABORT(255);
        }
    }

    singularity_message(ERROR, "We should never get here... Grrrrrr!\n");
    ABORT(255);
}
