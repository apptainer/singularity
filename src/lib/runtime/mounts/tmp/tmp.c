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
#include <unistd.h>
#include <stdlib.h>
#include <pwd.h>

#include "config.h"
#include "util/file.h"
#include "util/util.h"
#include "util/message.h"
#include "util/config_parser.h"
#include "util/registry.h"
#include "util/mountlist.h"

#include "../../runtime.h"


int _singularity_runtime_mount_tmp(struct mountlist *mountlist) {
    char *tmp_source;
    char *vartmp_source;

    if ( singularity_config_get_bool(MOUNT_TMP) <= 0 ) {
        singularity_message(VERBOSE, "Skipping tmp dir mounting (per config)\n");
        return(0);
    }

    if ( singularity_registry_get("CONTAIN") == NULL ) {
        tmp_source = strdup("/tmp");
        vartmp_source = strdup("/var/tmp");
    } else {
        char *tmpdirpath;
        if ( ( tmpdirpath = singularity_registry_get("WORKDIR") ) != NULL ) {
            if ( singularity_config_get_bool(USER_BIND_CONTROL) <= 0 ) {
                singularity_message(ERROR, "User bind control is disabled by system administrator\n");
                ABORT(5);
            }
            tmp_source = joinpath(tmpdirpath, "/tmp");
            vartmp_source = joinpath(tmpdirpath, "/var_tmp");
        } else {
            char *sessiondir = singularity_registry_get("SESSIONDIR");
            tmp_source = joinpath(sessiondir, "/tmp");
            vartmp_source = joinpath(sessiondir, "/var_tmp");
        }
        free(tmpdirpath);
    }

    if ( s_mkpath(tmp_source, 0755) < 0 ) {
        singularity_message(ERROR, "Could not create source /tmp directory %s: %s\n", tmp_source, strerror(errno));
        ABORT(255);
    }
    if ( is_dir(tmp_source) == 0 ) {
        singularity_message(VERBOSE, "Queuing bind mount of /tmp\n");
        mountlist_add(mountlist, strdup(tmp_source), strdup("/tmp"), NULL, MS_BIND|MS_NOSUID|MS_NODEV|MS_REC, NULL);
    } else {
        singularity_message(VERBOSE, "Could not mount host's /tmp directory (%s): does not exist\n", tmp_source);
    }

    if ( s_mkpath(vartmp_source, 0755) < 0 ) {
        singularity_message(ERROR, "Could not create source /var/tmp directory %s: %s\n", vartmp_source, strerror(errno));
        ABORT(255);
    }
    if ( is_dir(vartmp_source) == 0 ) {
        singularity_message(VERBOSE, "Queuing bind mount of /var/tmp\n");
        mountlist_add(mountlist, strdup(vartmp_source), strdup("/var/tmp"), NULL, MS_BIND|MS_NOSUID|MS_NODEV|MS_REC, NULL);
    } else {
        singularity_message(VERBOSE, "Could not mount host's /var/tmp directory (%s): does not exist\n", vartmp_source);
    }

    free(tmp_source);
    free(vartmp_source);

    return(0);
}
