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
#include "util/registry.h"

#include "./include.h"

#ifndef LIBEXECDIR
#error LIBEXECDIR is not defined
#endif


int bootstrap_init(int argc, char **argv) {
    char *builddef = singularity_registry_get("BUILDDEF");


    if ( strncmp(builddef, "docker://", 9) == 0 ) {
        char *bootstrap = joinpath(LIBEXECDIR, "/singularity/bootstrap-scripts/main-dockerhub.sh");

        singularity_message(INFO, "Building from DockerHub container\n");
        execl(bootstrap, bootstrap, NULL); // Flawfinder: ignore (this is necessary)

    } else if ( strncmp(builddef, "self", 4) == 0 ) {

        char *bootstrap = joinpath(LIBEXECDIR, "/singularity/bootstrap-scripts/main-deffile.sh");
        singularity_message(INFO, "Self clone with bootstrap definition recipe\n");

        if ( bootstrap_keyval_parse(builddef) != 0 ) {
            singularity_message(ERROR, "Failed parsing the bootstrap definition file: %s\n", singularity_registry_get("BUILDDEF"));
            ABORT(255);
        }
        execl(bootstrap, bootstrap, NULL); // Flawfinder: ignore (this is necessary)


    } else if ( is_file(builddef) == 0 ) {
        char *bootstrap = joinpath(LIBEXECDIR, "/singularity/bootstrap-scripts/main-deffile.sh");

        singularity_message(INFO, "Building from bootstrap definition recipe\n");
        if ( bootstrap_keyval_parse(builddef) != 0 ) {
            singularity_message(ERROR, "Failed parsing the bootstrap definition file: %s\n", singularity_registry_get("BUILDDEF"));
            ABORT(255);
        }
        execl(bootstrap, bootstrap, NULL); // Flawfinder: ignore (this is necessary)

    } else if ( builddef == NULL || builddef[0] == '\0' ) {
        char *bootstrap = joinpath(LIBEXECDIR, "/singularity/bootstrap-scripts/main-null.sh");

        singularity_message(INFO, "Running bootstrap with no recipe\n");
        execl(bootstrap, bootstrap, NULL); // Flawfinder: ignore (this is necessary)

    } else {
        singularity_message(ERROR, "Unsupported bootstrap definition format: '%s'\n", builddef);
        ABORT(255);

    }

    return(0);
}
