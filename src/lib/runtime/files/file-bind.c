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
#include <sys/mount.h>
#include <unistd.h>
#include <stdlib.h>

#include "config.h"
#include "util/file.h"
#include "util/util.h"
#include "util/message.h"
#include "util/privilege.h"
#include "util/mount.h"

#include "../runtime.h"


int container_file_bind(char *source, char *dest_path) {
    char *dest;
    char *containerdir = CONTAINER_FINALDIR;

    singularity_message(DEBUG, "Called file_bind(%s, %s()\n", source, dest_path);

    if ( containerdir == NULL ) {
        singularity_message(ERROR, "Failed to obtain container directory\n");
        ABORT(255);
    }

    dest = joinpath(containerdir, dest_path);

    if ( is_file(source) < 0 ) {
        singularity_message(WARNING, "Bind file source does not exist on host: %s\n", source);
        return(1);
    }

    if ( is_file(dest) < 0 ) {
        singularity_message(VERBOSE, "Skipping bind file, destination does not exist in container: %s\n", dest_path);
        return(0);
    }

    singularity_priv_escalate();
    singularity_message(VERBOSE, "Binding file '%s' to '%s'\n", source, dest);
    if ( singularity_mount(source, dest, NULL, MS_BIND|MS_NOSUID|MS_NODEV|MS_REC, NULL) < 0 ) {
        singularity_priv_drop();
        singularity_message(ERROR, "There was an error binding %s to %s: %s\n", source, dest, strerror(errno));
        ABORT(255);
    }
    if ( singularity_priv_userns_enabled() != 1 ) {
        if ( singularity_mount(NULL, dest, NULL, MS_BIND|MS_NOSUID|MS_NODEV|MS_REC|MS_REMOUNT, NULL) < 0 ) {
            singularity_priv_drop();
            singularity_message(ERROR, "There was an error remounting %s to %s: %s\n", source, dest, strerror(errno));
            ABORT(255);
        }
    }
    singularity_priv_drop();

    return(0);
}
