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

#define _GNU_SOURCE
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
#include "util/config_parser.h"
#include "util/registry.h"



int _singularity_runtime_autofs(void) {
    char *source;
    int autofs_fd;

    const char **tmp_config_string_list = singularity_config_get_value_multi(AUTOFS_BUG_PATH);

    if ( strlength(*tmp_config_string_list, 1) == 0 ) {
        singularity_message(VERBOSE, "No autofs bug path in configuration, skipping\n");
        return(0);
    }

    singularity_message(VERBOSE, "Autofs bug path requested\n");

    while ( *tmp_config_string_list != NULL ) {
        source = strdup(*tmp_config_string_list);
        tmp_config_string_list++;
        chomp(source);

        singularity_message(VERBOSE2, "Autofs bug fix for directory %s\n", source);

        if ( is_dir(source) < 0 ) {
            singularity_message(WARNING, "Autofs bug path %s is not a directory\n", source);
            continue;
        }

        autofs_fd = open(source, O_RDONLY);
        if ( autofs_fd < 0 ) {
            singularity_message(WARNING, "Failed to open directory '%s'\n", source);
            continue;
        }

        if ( fcntl(autofs_fd, F_SETFD, FD_CLOEXEC) != 0 ) {
            singularity_message(WARNING, "Failed to set FD_CLOEXEC on directory '%s'\n", source);
            continue;
        }
    }

    return(0);
}

