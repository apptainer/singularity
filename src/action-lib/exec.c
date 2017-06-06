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


void action_exec(int argc, char **argv) {

    if ( argc <= 1 ) {
        singularity_message(ERROR, "No program name to exec\n");
        ABORT(255);
    }

    singularity_message(DEBUG, "Checking for: /.singularity.d/actions/exec\n");
    if ( is_exec("/.singularity.d/actions/exec") == 0 ) {
        singularity_message(VERBOSE, "Exec'ing /.singularity.d/actions/exec\n");
        if ( execv("/.singularity.d/actions/exec", argv) < 0 ) { // Flawfinder: ignore
            singularity_message(ERROR, "Failed to execv() /.singularity.d/actions/exec: %s\n", strerror(errno));
        }
    }

    singularity_message(DEBUG, "Checking for: /.exec\n");
    if ( is_exec("/.exec") == 0 ) {
        singularity_message(VERBOSE, "Exec'ing /.exec\n");
        if ( execv("/.exec", argv) < 0 ) { // Flawfinder: ignore
            singularity_message(ERROR, "Failed to execv() /.exec: %s\n", strerror(errno));
        }
    }

    singularity_message(WARNING, "Container does not have an exec helper script, calling '%s' directly\n", argv[1]);
    if ( execvp(argv[1], &argv[1]) < 0 ) { // Flawfinder: ignore
        singularity_message(ERROR, "Failed to execvp() %s: %s\n", argv[1], strerror(errno));
        ABORT(255);
    }

    singularity_message(ERROR, "We should never get here... Grrrrrr!\n");
    ABORT(255);
}
