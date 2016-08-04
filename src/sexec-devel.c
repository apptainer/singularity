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

#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>

#include "config.h"
#include "config_parser.h"
#include "message.h"
#include "util.h"
#include "privilege.h"
#include "sessiondir.h"
#include "singularity.h"


int main(int argc, char **argv) {
    char *sessiondir;
    char *image = getenv("SINGULARITY_IMAGE");

    if ( image == NULL ) {
        message(ERROR, "SINGULARITY_IMAGE not defined!\n");
        ABORT(1);
    }

    priv_init();
    config_open("/etc/singularity/singularity.conf");

    message(INFO, "SINGULARITY_IMAGE = '%s'\n", image);

    sessiondir = singularity_sessiondir(image);

    message(INFO, "Sessiondir = '%s'\n", sessiondir);
    
    printf("Calling singularity_init()\n");

    singularity_ns_init();

    return(0);

}
