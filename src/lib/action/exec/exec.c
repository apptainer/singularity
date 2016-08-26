/* 
 * Copyright (c) 2015-2016, Gregory M. Kurtzer. All rights reserved.
 * 
 * “Singularity” Copyright (c) 2016, The Regents of the University of California,
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
#include "message.h"
#include "lib/privilege.h"


//TODO: Add backwards compatibility
void action_exec_do(int argc, char **argv) {
    message(VERBOSE, "Exec'ing /.exec\n");

    if ( is_exec("/.exec") == 0 ) {
        if ( execv("/.exec", argv) < 0 ) {
            message(ERROR, "Failed to execv() /.exec\n");
        }
    }

    if ( execvp(argv[1], &argv[1]) < 0 ) {
        message(ERROR, "Failed to execvp() /.exec\n");
        ABORT(255);
    }


    message(ERROR, "We should never get here... Grrrrrr!\n");
    ABORT(255);
}
