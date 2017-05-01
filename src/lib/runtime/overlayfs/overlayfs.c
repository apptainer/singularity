/* 
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
#include <sys/mount.h>
#include <unistd.h>
#include <stdlib.h>
#include <libgen.h>
#include <linux/limits.h>

#include "util/file.h"
#include "util/util.h"
#include "util/registry.h"
#include "util/message.h"
#include "util/config_parser.h"
#include "util/privilege.h"

#include "../runtime.h"


int _singularity_runtime_overlayfs(void) {
    char *rootfs_source = singularity_runtime_rootfs(NULL);
    char *container_dir = strdup(singularity_config_get_value(CONTAINER_DIR));
    char *mount_final   = joinpath(container_dir, "/final");
    int overlay_enabled = 0;

    singularity_message(DEBUG, "Checking if overlayfs should be used\n");
    if ( singularity_config_get_bool(ENABLE_OVERLAY) <= 0 ) {
        singularity_message(VERBOSE3, "Not enabling overlayFS via configuration\n");
    } else if ( singularity_registry_get("DISABLE_OVERLAYFS") != NULL ) {
        singularity_message(VERBOSE3, "Not enabling overlayFS via environment\n");
    } else if ( singularity_registry_get("WRITABLE") != NULL ) {
        singularity_message(VERBOSE3, "Not enabling overlayFS, image mounted writablable\n");
    } else {
#ifdef SINGULARITY_OVERLAYFS
        char *overlay_mount = joinpath(container_dir, "/overlay");
        char *overlay_upper = joinpath(container_dir, "/overlay/upper");
        char *overlay_work  = joinpath(container_dir, "/overlay/work");
        int overlay_options_len = strlength(rootfs_source, PATH_MAX) + strlength(overlay_upper, PATH_MAX) + strlength(overlay_work, PATH_MAX) + 50;
        char *overlay_options = (char *) malloc(overlay_options_len);

        singularity_message(DEBUG, "OverlayFS enabled by host build\n");

        snprintf(overlay_options, overlay_options_len, "lowerdir=%s,upperdir=%s,workdir=%s", rootfs_source, overlay_upper, overlay_work); // Flawfinder: ignore

        singularity_priv_escalate();
        singularity_message(DEBUG, "Creating top level overlay mount directory: %s\n", overlay_mount);
        if ( s_mkpath(overlay_mount, 0755) < 0 ) {
            singularity_message(ERROR, "Could not create overlay_mount directory %s: %s\n", overlay_mount, strerror(errno));
            ABORT(255);
        }

        singularity_message(DEBUG, "Mounting overlay tmpfs: %s\n", overlay_mount);
        if ( mount("tmpfs", overlay_mount, "tmpfs", MS_NOSUID, "size=1m") < 0 ){
            singularity_message(ERROR, "Failed to mount overlay tmpfs %s: %s\n", overlay_mount, strerror(errno));
            ABORT(255);
        }

        singularity_message(DEBUG, "Creating upper overlay directory: %s\n", overlay_upper);
        if ( s_mkpath(overlay_upper, 0755) < 0 ) {
            singularity_message(ERROR, "Failed creating upper overlay directory %s: %s\n", overlay_upper, strerror(errno));
            ABORT(255);
        }

        singularity_message(DEBUG, "Creating overlay work directory: %s\n", overlay_work);
        if ( s_mkpath(overlay_work, 0755) < 0 ) {
            singularity_message(ERROR, "Failed creating overlay work directory %s: %s\n", overlay_work, strerror(errno));
            ABORT(255);
        }

        singularity_message(DEBUG, "Creating mount_final directory: %s\n", mount_final);
        if ( s_mkpath(mount_final, 0755) < 0 ) {
            singularity_message(ERROR, "Failed creating mount_final directory %s: %s\n", mount_final, strerror(errno));
            ABORT(255);
        }

        singularity_message(VERBOSE, "Mounting overlay with options: %s\n", overlay_options);
        if ( mount("overlay", mount_final, "overlay", MS_NOSUID, overlay_options) < 0 ){
            singularity_message(ERROR, "Could not mount overlayFS: %s\n", strerror(errno));
            ABORT(255); 
        }
        singularity_priv_drop();

        free(overlay_mount);
        free(overlay_upper);
        free(overlay_options);

        overlay_enabled = 1;
        singularity_registry_set("OVERLAYFS_ENABLED", "1");
#else
        singularity_message(VERBOSE, "OverlayFS not supported by host build\n");
#endif
    }

    if ( overlay_enabled != 1 ) {
        singularity_priv_escalate();
        singularity_message(DEBUG, "Creating mount_final directory: %s\n", mount_final);
        if ( s_mkpath(mount_final, 0755) < 0 ) {
            singularity_message(ERROR, "Failed creating mount_final directory %s: %s\n", mount_final, strerror(errno));
            ABORT(255);
        }

        singularity_message(VERBOSE3, "Binding the ROOTFS_SOURCE to OVERLAY_FINAL (%s->%s)\n", rootfs_source, mount_final);
        if ( mount(rootfs_source, mount_final, NULL, MS_BIND|MS_NOSUID|MS_REC, NULL) < 0 ) {
            singularity_message(ERROR, "There was an error binding the container to path %s: %s\n", mount_final, strerror(errno));
            ABORT(255);
        }
        singularity_priv_drop();
    }

    // If we got here, then we now set the runtime containerdir to our new mount point
    singularity_message(VERBOSE2, "Updating the containerdir to: %s\n", mount_final);
    singularity_runtime_rootfs(mount_final);

    return(0);
}
