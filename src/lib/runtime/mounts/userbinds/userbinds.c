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
#include <unistd.h>
#include <stdlib.h>
#include <libgen.h>

#include "config.h"
#include "util/file.h"
#include "util/util.h"
#include "util/message.h"
#include "util/config_parser.h"
#include "util/registry.h"
#include "util/mountlist.h"

#include "../../runtime.h"


int _singularity_runtime_mount_userbinds(struct mountlist *mountlist) {
    char *bind_path_string;

    singularity_message(DEBUG, "Checking for environment variable 'SINGULARITY_BINDPATH'\n");
    if ( ( bind_path_string = singularity_registry_get("BINDPATH") ) != NULL ) {

        singularity_message(DEBUG, "Checking for 'user bind control' in config\n");
        if ( singularity_config_get_bool(USER_BIND_CONTROL) <= 0 ) {
            singularity_message(WARNING, "Ignoring user bind request: user bind control is disabled by system administrator\n");
            return(0);
        }

        singularity_message(DEBUG, "Parsing SINGULARITY_BINDPATH for user-specified bind mounts.\n");
        char *outside_token = NULL;
        char *inside_token = NULL;
        char *current = strtok_r(bind_path_string, ",", &outside_token);

        while ( current != NULL ) {
            unsigned long read_only = 0;
            char *source = strtok_r(current, ":", &inside_token);
            char *dest = strtok_r(NULL, ":", &inside_token);
            char *opts = strtok_r(NULL, ":", &inside_token);

            current = strtok_r(NULL, ",", &outside_token);

            if ( dest == NULL ) {
                dest = source;
            }

            singularity_message(DEBUG, "Found bind: %s -> container:%s\n", source, dest);

            if ( opts != NULL ) {
                if ( strcmp(opts, "rw") == 0 ) {
                    // This is the default
                } else if ( strcmp(opts, "ro") == 0 ) {
                    read_only = MS_RDONLY;
                } else {
                    singularity_message(WARNING, "Not mounting requested bind point, invalid mount option %s: %s\n", opts, dest);
                    continue;
                }
            }


            singularity_message(VERBOSE, "Queuing bind mount of '%s' to '%s'\n", source, dest);
            mountlist_add(mountlist, strdup(source), strdup(dest), NULL, MS_BIND|MS_NOSUID|MS_NODEV|MS_REC|read_only, 0);
        }

        free(bind_path_string);

        singularity_message(DEBUG, "Unsetting environment variable 'SINGULARITY_BINDPATH'\n");
        unsetenv("SINGULARITY_BINDPATH");
    } else {
        singularity_message(DEBUG, "No user bind mounts specified.\n");
    }
    return(0);
}

