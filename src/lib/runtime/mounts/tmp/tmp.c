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
#include "util/privilege.h"
#include "util/config_parser.h"
#include "util/registry.h"
#include "util/mount.h"

#include "../../runtime.h"


int _singularity_runtime_mount_tmp(void) {
    char *container_dir = CONTAINER_FINALDIR;
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

#ifndef SINGULARITY_NO_NEW_PRIVS
            singularity_message(WARNING, "Not mounting workdir: host does not support PR_SET_NO_NEW_PRIVS\n");
            ABORT(255);
#endif

            tmp_source = joinpath(tmpdirpath, "/tmp");
            vartmp_source = joinpath(tmpdirpath, "/var_tmp");
        } else {
            char *sessiondir = singularity_registry_get("SESSIONDIR");
            tmp_source = joinpath(sessiondir, "/tmp");
            vartmp_source = joinpath(sessiondir, "/var_tmp");
        }
        free(tmpdirpath);
    }

    if ( check_mounted("/tmp") < 0 ) {
        if ( s_mkpath(tmp_source, 0755) < 0 ) {
            singularity_message(ERROR, "Could not create source /tmp directory %s: %s\n", tmp_source, strerror(errno));
            ABORT(255);
        }
        if ( is_dir(tmp_source) == 0 ) {
            if ( is_dir(joinpath(container_dir, "/tmp")) == 0 ) {
                singularity_priv_escalate();
                singularity_message(VERBOSE, "Mounting directory: /tmp\n");
                if ( singularity_mount(tmp_source, joinpath(container_dir, "/tmp"), NULL, MS_BIND|MS_NOSUID|MS_NODEV|MS_REC, NULL) < 0 ) {
                    singularity_message(ERROR, "Failed to mount %s -> /tmp: %s\n", tmp_source, strerror(errno));
                    ABORT(255);
                }
                if ( singularity_priv_userns_enabled() != 1 ) {
                    if ( singularity_mount(NULL, joinpath(container_dir, "/tmp"), NULL, MS_BIND|MS_NOSUID|MS_NODEV|MS_REC|MS_REMOUNT, NULL) < 0 ) {
                        singularity_message(ERROR, "Failed to remount /tmp: %s\n", strerror(errno));
                        ABORT(255);
                    }
                }
                singularity_priv_drop();
            } else {
                singularity_message(VERBOSE, "Could not mount container's /tmp directory: does not exist\n");
            }
        } else {
            singularity_message(VERBOSE, "Could not mount host's /tmp directory (%s): does not exist\n", tmp_source);
        }
    } else {
        singularity_message(VERBOSE, "Not mounting '/tmp', already mounted\n");
    }

    if ( check_mounted("/var/tmp") < 0 ) {
        if ( s_mkpath(vartmp_source, 0755) < 0 ) {
            singularity_message(ERROR, "Could not create source /var/tmp directory %s: %s\n", vartmp_source, strerror(errno));
            ABORT(255);
        }
        if ( is_dir(vartmp_source) == 0 ) {
            if ( is_dir(joinpath(container_dir, "/var/tmp")) == 0 ) {
                singularity_priv_escalate();
                singularity_message(VERBOSE, "Mounting directory: /var/tmp\n");
                if ( singularity_mount(vartmp_source, joinpath(container_dir, "/var/tmp"), NULL, MS_BIND|MS_NOSUID|MS_NODEV|MS_REC, NULL) < 0 ) {
                    singularity_message(ERROR, "Failed to mount %s -> /var/tmp: %s\n", vartmp_source, strerror(errno));
                    ABORT(255);
                }
                if ( singularity_priv_userns_enabled() != 1 ) {
                    if ( singularity_mount(NULL, joinpath(container_dir, "/var/tmp"), NULL, MS_BIND|MS_NOSUID|MS_REC|MS_NODEV|MS_REMOUNT, NULL) < 0 ) {
                        singularity_message(ERROR, "Failed to remount /var/tmp: %s\n", strerror(errno));
                        ABORT(255);
                    }
                }
                singularity_priv_drop();
            } else {
                singularity_message(VERBOSE, "Could not mount container's /var/tmp directory: does not exist\n");
            }
        } else {
            singularity_message(VERBOSE, "Could not mount host's /var/tmp directory (%s): does not exist\n", vartmp_source);
        }
    } else {
        singularity_message(VERBOSE, "Not mounting '/var/tmp', already mounted\n");
    }

    free(tmp_source);
    free(vartmp_source);

    return(0);
}
