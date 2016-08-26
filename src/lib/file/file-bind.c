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
#include <sys/mount.h>
#include <unistd.h>
#include <stdlib.h>

#include "util/file.h"
#include "util/util.h"
#include "lib/message.h"
#include "lib/privilege.h"
#include "lib/sessiondir.h"
#include "lib/singularity.h"


int container_file_bind(char *file, char *dest_path) {
    char *source;
    char *dest;
    char *containerdir = singularity_rootfs_dir();
    char *sessiondir = singularity_sessiondir_get();

    singularity_message(DEBUG, "Called file_bind(%s, %s()\n", file, dest_path);

    if ( containerdir == NULL ) {
        singularity_message(ERROR, "Failed to obtain container directory\n");
        ABORT(255);
    }

    if ( sessiondir == NULL ) {
        singularity_message(ERROR, "Failed to obtain session directory\n");
        ABORT(255);
    }

    source = joinpath(sessiondir, file);
    dest = joinpath(containerdir, dest_path);

    if ( is_file(source) < 0 ) {
        singularity_message(ERROR, "Bind file source does not exist: %s\n", source);
        ABORT(255);
    }

    if ( is_file(dest) < 0 ) {
        singularity_message(ERROR, "Bind file source does not exist: %s\n", dest);
        ABORT(255);
    }

    singularity_priv_escalate();
    singularity_message(VERBOSE, "Binding file '%s' to '%s'\n", source, dest);
    if ( mount(source, dest, NULL, MS_BIND|MS_NOSUID|MS_REC, NULL) < 0 ) {
        singularity_priv_drop();
        singularity_message(ERROR, "There was an error binding %s to %s: %s\n", source, dest, strerror(errno));
        ABORT(255);
    }
    singularity_priv_drop();

    return(0);
}
