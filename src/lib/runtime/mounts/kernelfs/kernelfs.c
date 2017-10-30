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
#include "util/config_parser.h"
#include "util/registry.h"
#include "util/mount.h"

#include "../../runtime.h"
#include "../../ns/ns.h"


int _singularity_runtime_mount_kernelfs(void) {
    char *container_dir = CONTAINER_FINALDIR;

    // Mount /proc if we are configured
    singularity_message(DEBUG, "Checking configuration file for 'mount proc'\n");
    if ( singularity_config_get_bool(MOUNT_PROC) > 0 ) {
        if ( is_dir(joinpath(container_dir, "/proc")) == 0 ) {
            if ( singularity_registry_get("PIDNS_ENABLED") == NULL ) {
                singularity_priv_escalate();
                singularity_message(VERBOSE, "Bind-mounting host /proc\n");
                if ( singularity_mount("/proc", joinpath(container_dir, "/proc"), NULL, MS_BIND | MS_NOSUID | MS_REC, NULL) < 0 ) {
                    singularity_message(ERROR, "Could not bind-mount host /proc into container: %s\n", strerror(errno));
                    ABORT(255);
                }
                singularity_priv_drop();
            } else {
                singularity_priv_escalate();
                singularity_message(VERBOSE, "Mounting new procfs\n");
                if ( singularity_mount("proc", joinpath(container_dir, "/proc"), "proc", MS_NOSUID, NULL) < 0 ) {
                    singularity_message(ERROR, "Could not mount new procfs into container: %s\n", strerror(errno));
                    ABORT(255);
                }
                singularity_priv_drop();
            }
        } else {
            singularity_message(WARNING, "Not mounting /proc, container has no bind directory\n");
        }
    } else {
        singularity_message(VERBOSE, "Skipping /proc mount\n");
    }


    // Mount /sys if we are configured
    singularity_message(DEBUG, "Checking configuration file for 'mount sys'\n");
    if ( singularity_config_get_bool(MOUNT_SYS) > 0 ) {
        if ( is_dir(joinpath(container_dir, "/sys")) == 0 ) {
            if ( singularity_priv_userns_enabled() == 1 ) {
                singularity_priv_escalate();
                singularity_message(VERBOSE, "Mounting /sys\n");
                if ( singularity_mount("/sys", joinpath(container_dir, "/sys"), NULL, MS_BIND | MS_NOSUID | MS_REC, NULL) < 0 ) {
                    singularity_message(ERROR, "Could not mount /sys into container: %s\n", strerror(errno));
                    ABORT(255);
                }
                singularity_priv_drop();
            } else {
                singularity_priv_escalate();
                singularity_message(VERBOSE, "Mounting /sys\n");
                if ( singularity_mount("sysfs", joinpath(container_dir, "/sys"), "sysfs", MS_NOSUID, NULL) < 0 ) {
                    singularity_message(ERROR, "Could not mount /sys into container: %s\n", strerror(errno));
                    ABORT(255);
                }
                singularity_priv_drop();
            }
        } else {
            singularity_message(WARNING, "Not mounting /sys, container has no bind directory\n");
        }
    } else {
        singularity_message(VERBOSE, "Skipping /sys mount\n");
    }

    return(0);
}
