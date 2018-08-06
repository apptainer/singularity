/* 
 * Copyright (c) 2017-2018, SyLabs, Inc. All rights reserved.
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
#include <sys/mount.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <unistd.h>
#include <stdlib.h>
#include <libgen.h>
#include <linux/limits.h>

#include "config.h"
#include "util/file.h"
#include "util/util.h"
#include "util/message.h"
#include "util/privilege.h"
#include "util/config_parser.h"
#include "util/registry.h"
#include "util/mountlist.h"

#include "../../runtime.h"


int _singularity_runtime_mount_scratch(struct mountlist *mountlist) {
    char *scratchdir_path;
    char *tmpdir_path;
    char *sourcedir_path;

    singularity_message(DEBUG, "Getting SINGULARITY_SCRATCHDIR from environment\n");
    if ( ( scratchdir_path = singularity_registry_get("SCRATCHDIR") ) == NULL ) {
        singularity_message(DEBUG, "Not mounting scratch directory: Not requested\n");
        return(0);
    }

    singularity_message(DEBUG, "Checking configuration file for 'user bind control'\n");
    if ( singularity_config_get_bool(USER_BIND_CONTROL) <= 0 ) {
        singularity_message(VERBOSE, "Not mounting scratch: user bind control is disabled by system administrator\n");
        return(0);
    }

    singularity_message(DEBUG, "Checking SINGULARITY_WORKDIR from environment\n");
    if ( ( tmpdir_path = singularity_registry_get("WORKDIR") ) == NULL ) {
        if ( ( tmpdir_path = singularity_registry_get("SESSIONDIR") ) == NULL ) {
            singularity_message(ERROR, "Could not identify a suitable temporary directory for scratch\n");
            return(0);
        }
    }

    sourcedir_path = joinpath(tmpdir_path, "/scratch");

    free(tmpdir_path);

    char *outside_token = NULL;
    char *current = strtok_r(scratchdir_path, ",", &outside_token);

    while ( current != NULL ) {
        char *current_copy = strdup(current);
        char *full_sourcedir_path = joinpath(sourcedir_path, basename(current_copy));
        free(current_copy);

        if ( container_mkpath_nopriv(full_sourcedir_path, 0750) < 0 ) {
             singularity_message(ERROR, "Could not create scratch working directory %s: %s\n", full_sourcedir_path, strerror(errno));
             ABORT(255);
        }

        singularity_message(VERBOSE, "Queuing bind mount of '%s' to '%s'\n", full_sourcedir_path, current);
        mountlist_add(mountlist, full_sourcedir_path, strdup(current), NULL, MS_BIND|MS_NOSUID|MS_NODEV|MS_REC, 0);

        current = strtok_r(NULL, ",", &outside_token);

        // Skip empty directories.
        while ( current && !strlength(current, 1024) ) {
            current = strtok_r(NULL, ",", &outside_token);
        }
    }

    free(scratchdir_path);
    return(0);
}

