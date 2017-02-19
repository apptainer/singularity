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
#include "util/fork.h"

#include "util/registry.h"

#ifndef LIBEXECDIR
#error LIBEXECDIR is not defined
#endif


int bootstrap_driver(void) {
    char *bootstrap_pre;
    char *bootstrap_env;
    char *bootstrap_post;
    char *bootstrap_driver;
    char *driver_script;
    char *driver = singularity_registry_get("DRIVER");
    char *driverproc[2];

    if ( driver == NULL ) {
        singularity_message(ERROR, "No 'BootStrap' key/value defined in definition file\n");
        ABORT(255);
    }

    bootstrap_pre = joinpath(LIBEXECDIR, "/singularity/bootstrap-scripts/pre.sh");
    bootstrap_env = joinpath(LIBEXECDIR, "/singularity/bootstrap-scripts/env.sh");
    bootstrap_post = joinpath(LIBEXECDIR, "/singularity/bootstrap-scripts/post.sh");

    driver_script = strjoin(driver, ".sh");
    bootstrap_driver = joinpath(LIBEXECDIR, strjoin("/singularity/bootstrap-scripts/driver-", driver_script));

    driverproc[1] = NULL;

    if ( is_file(bootstrap_driver) != 0 ) {
        singularity_message(ERROR, "Bootstrap driver not supported: %s\n", bootstrap_driver);
        ABORT(255);
    }

    setenv("SINGULARITY_libexecdir", LIBEXECDIR, 1);

    driverproc[0] = bootstrap_pre;

    if ( singularity_fork_exec(driverproc) != 0 ) {
        ABORT(255);
    }

    driverproc[0] = bootstrap_env;

    if ( singularity_fork_exec(driverproc) != 0 ) {
        ABORT(255);
    }

    driverproc[0] = bootstrap_driver;

    if ( singularity_fork_exec(driverproc) != 0 ) {
        ABORT(255);
    }

    driverproc[0] = bootstrap_post;

    if ( singularity_fork_exec(driverproc) != 0 ) {
        ABORT(255);
    }

    return(0);
}

