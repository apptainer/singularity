/* 
 * Copyright (c) 2016, Michael W. Bauer. All rights reserved.
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
#include <sys/types.h>
#include <limits.h>
#include <unistd.h>
#include <stdlib.h>

#include "util/file.h"
#include "util/util.h"
#include "lib/message.h"
#include "lib/rootfs/rootfs.h"


int singularity_file_entrypoint(char *entrypoint_name) {
    singularity_message(DEBUG, "Copying entrypoint file: %s\n", entrypoint_name);
    int retval = 0;
    char *helper_shell;
    char *rootfs_path = singularity_rootfs_dir();
    char *dest_path = joinpath(rootfs_path, strjoin("/.", entrypoint_name));
    char *entrypoint = filecat(strjoin(LIBEXECDIR "/singularity/defaults/", entrypoint_name));

    if ( is_file(joinpath(rootfs_path, "/bin/bash")) == 0 ) {
        helper_shell = strdup("#!/bin/bash");
    } else {
        helper_shell = strdup("#!/bin/sh");
    }

    retval += fileput(dest_path, strjoin(helper_shell, entrypoint));
    retval += chmod(dest_path, 0755);

    free(helper_shell);
    free(rootfs_path);
    free(dest_path);
    free(entrypoint);

    return(retval);
}
