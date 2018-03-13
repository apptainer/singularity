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
#include <sys/stat.h>
#include <sys/types.h>
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

#include "../file-bind.h"
#include "../../runtime.h"


int _singularity_runtime_files_11_user_defined_SINGULARITYENV_PATH(void) {
    char *tmp_file;
    char *containerdir = CONTAINER_FINALDIR;
    char *tmpdir = singularity_registry_get("SESSIONDIR");
    char *user_add_path = singularity_registry_get("USER_DEFINED_PREPEND");

    singularity_message(DEBUG, "Called _singularity_runtime_files_11-user_defined_SINGULARITYENV_PATH()\n");

    if ( user_add_path == NULL ) {
        singularity_message(VERBOSE2, "No user defined SINGULARITYENV_PATH found.\n");
        return 0; 
    }

    if ( tmpdir == NULL ) {
        singularity_message(ERROR, "Failed to obtain session directory\n");
        ABORT(255);
    }

    tmp_file = joinpath(tmpdir, "/11-user_defined_SINGULARITYENV_PATH.sh");

    singularity_message(VERBOSE2, "Creating empty /.singularity.d/env/11-user_defined_SINGULARITYENV_PATH.sh in %s\n", containerdir);
    singularity_priv_escalate();
    if ( ( fileput(joinpath(containerdir, "/.singularity.d/env/11-user_defined_SINGULARITYENV_PATH.sh"), "") ) !=0 ) {
        singularity_priv_drop();
        singularity_message(ERROR, "Failed to create empty /.singularity.d/env/11-user_defined_SINGULARITYENV_PATH.sh in containerdir: %s\n", strerror(errno));
        ABORT(255);
    }
    singularity_priv_drop();

    char *path_str = malloc(19 + strlen(user_add_path) + 1);
    if ( path_str == NULL ) {
        singularity_message(ERROR, "Failed to allocate memory for user defined PATH\n");
        ABORT(255);
    }
    strcpy(path_str, "export PATH=");
    strcat(path_str, user_add_path);
    strcat(path_str, ":$PATH\n\0"); 

    singularity_message(VERBOSE2, "Creating template of /.singularity.d/env/11-user_defined_SINGULARITYENV_PATH.sh\n");
    if ( ( fileput(tmp_file, path_str) ) !=0 ) {
        singularity_message(ERROR, "Failed creating template 11-user_defined_SINGULARITYENV_PATH.sh file in tmpdir: %s\n", strerror(errno));
        ABORT(255);
    }

    free(path_str);

    container_file_bind(tmp_file, "/.singularity.d/env/11-user_defined_SINGULARITYENV_PATH.sh");

    return(0);
}
