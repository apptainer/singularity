/* 
 * Copyright (c) 2017-2018, SyLabs, Inc. All rights reserved.
 * 
 * This software is licensed under a customized 3-clause BSD license.  Please
 * consult LICENSE file distributed with the sources of this project regarding
 * your rights to use or distribute this software.
 * 
*/

#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <string.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/mount.h>
#include <limits.h>
#include <unistd.h>
#include <stdlib.h>
#include <grp.h>
#include <pwd.h>

#include "config.h"
#include "util/file.h"
#include "util/util.h"
#include "util/config_parser.h"
#include "util/message.h"
#include "util/privilege.h"
#include "util/registry.h"
#include "util/mount.h"

#include "../file-bind.h"
#include "../../runtime.h"


int _singularity_runtime_files_11_user_defined_SINGULARITYENV_PATH(void) {
    char *container = CONTAINER_FINALDIR;
    char *host = singularity_registry_get("SESSIONDIR");
    char *containerenv = joinpath(container, "/.singularity.d/env");
    char *hostenv = joinpath(host, "/env");
    char *udsep = joinpath(host, "/env/11-user_defined_SINGULARITYENV_PATH.sh");
    char *udsep_var = singularity_registry_get("USER_DEFINED_PREPEND");

    singularity_message(DEBUG, "Called _singularity_runtime_files_11-user_defined_SINGULARITYENV_PATH()\n");

    if ( udsep_var == NULL ) {
        singularity_message(VERBOSE2, "No user defined SINGULARITYENV_PATH found.\n");
        return 0; 
    }

    if ( host == NULL ) {
        singularity_message(ERROR, "Failed to obtain session directory\n");
        ABORT(255);
    }

    // for systems without overlay support, first copy env dir from container
    singularity_message(VERBOSE2, "Copying .singularity.d/env from %s to %s\n", container, host);
    if ( ( copy_dir_r(containerenv, host ) ) !=0 ) {
        singularity_message(ERROR, "Failed to copy .singularity.d/env from %s to %s\n", container, host);
        ABORT(255);
    } 

    // create the string that should go into the new meta-data file
    char *udsep_str = malloc(19 + strlen(udsep_var) + 1);
    if ( udsep_str == NULL ) {
        singularity_message(ERROR, "Failed to allocate memory for user defined PATH\n");
        ABORT(255);
    }
    strcpy(udsep_str, "export PATH=");
    strcat(udsep_str, udsep_var);
    strcat(udsep_str, ":$PATH\n\0"); 

    // create the new meta-data file on the host in the env dir
    singularity_message(VERBOSE2, "Creating template of %s\n", udsep);
    if ( ( fileput(udsep, udsep_str) ) !=0 ) {
        free(udsep_str);
        singularity_message(ERROR, "Failed creating template %s: %s\n", udsep, strerror(errno));
        ABORT(255);
    }

    free(udsep_str);

    // mount the env dir on the host over the env dir in the container
    singularity_message(VERBOSE, "Mounting directory '%s' to '%s'\n", hostenv, containerenv);
    if ( singularity_mount(hostenv, containerenv, NULL, MS_BIND|MS_NOSUID|MS_NODEV|MS_REC, NULL) < 0 ) {
            singularity_message(ERROR, "There was an error binding %s to %s: %s\n", hostenv, containerenv, strerror(errno));
            ABORT(255);
    }

    return(0);
}
