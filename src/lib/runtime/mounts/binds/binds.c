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


int _singularity_runtime_mount_binds(struct mountlist *mountlist) {
    char *tmp_config_string;
    char *source = NULL;
    char *dest = NULL;

    if ( singularity_registry_get("CONTAIN") != NULL ) {
        singularity_message(DEBUG, "Skipping bind mounts as contain was requested\n");
        return(0);
    }

    singularity_message(DEBUG, "Checking configuration file for 'bind path'\n");
    const char **tmp_config_string_list = singularity_config_get_value_multi(BIND_PATH);
    if ( strlength(*tmp_config_string_list, 1) == 0 ) {
        return(0);
    }
    while ( *tmp_config_string_list != NULL ) {
        tmp_config_string = strdup(*tmp_config_string_list);
        tmp_config_string_list++;
        if (source != NULL)
            free(source);
        if (dest != NULL)
            free(dest);
        source = strtok(tmp_config_string, ":");
        dest = strtok(NULL, ":");
        source = strdup(source);
        chomp(source);
        if ( dest == NULL ) {
            dest = strdup(source);
        } else {
            dest = strdup(dest);
            chomp(dest);
        }
        free(tmp_config_string);

        singularity_message(VERBOSE2, "Found 'bind path' = %s, %s\n", source, dest);

        if ( ( is_file(source) < 0 ) && ( is_dir(source) < 0 ) ) {
            singularity_message(WARNING, "Non existent 'bind path' source: '%s'\n", source);
            continue;
        }

        singularity_message(VERBOSE, "Queuing bind mount of '%s' to '%s'\n", source, dest);
        mountlist_add(mountlist, source, dest, NULL, MS_BIND|MS_NOSUID|MS_NODEV|MS_REC, NULL);
        source = NULL;
        dest = NULL;
    }

    if (source != NULL)
        free(source);
    if (dest != NULL)
        free(dest);

    return(0);
}

