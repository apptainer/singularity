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

#include "file.h"
#include "util.h"
#include "message.h"
#include "privilege.h"


//TODO: Put in backwards compatibility for previous
void action_shell_do(int argc, char **argv) {
    message(VERBOSE, "Starting shell...\n");

    if ( is_exec("/.shell") == 0 ) {
        message(DEBUG, "Found container's /.shell, executing that\n");
        if ( execv("/.shell", argv) < 0 ) {
            message(ERROR, "Failed to execv() /.shell, continuing to /bin/sh\n");
        }
    }
    if ( is_exec("/bin/sh") == 0 ) {
        message(DEBUG, "Exec'ing /bin/sh\n");
        argv[0] = strdup("/bin/sh");
        if ( execv("/bin/sh", argv) < 0 ) {
            message(ERROR, "Failed to execv() /bin/sh\n");
            ABORT(255);
        }
    }

    message(ERROR, "We should never get here... Grrrrrr!\n");
    ABORT(255);
}
