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
    char *container_dir = joinpath(LOCALSTATEDIR, "/singularity/mnt");

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
        char *overlay_final = joinpath(container_dir, "/overlay/final");
        int overlay_options_len = strlength(rootfs_source, PATH_MAX) + strlength(overlay_upper, PATH_MAX) + strlength(overlay_work, PATH_MAX) + 50;
        char *overlay_options = (char *) malloc(overlay_options_len);

        singularity_message(DEBUG, "OverlayFS enabled by host build\n");

        singularity_message(DEBUG, "Setting up overlay mount options\n");
        snprintf(overlay_options, overlay_options_len, "lowerdir=%s,upperdir=%s,workdir=%s", rootfs_source, overlay_upper, overlay_work); // Flawfinder: ignore

        singularity_message(DEBUG, "Checking for existance of overlay directory: %s\n", overlay_mount);
        if ( is_dir(overlay_mount) < 0 ) {
            singularity_message(ERROR, "Overlay mount directory does not exist: %s\n", overlay_mount);
            ABORT(255);
        }

        singularity_priv_escalate();
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

        singularity_message(DEBUG, "Creating overlay_final directory: %s\n", overlay_final);
        if ( s_mkpath(overlay_final, 0755) < 0 ) {
            singularity_message(ERROR, "Failed creating overlay_final directory %s: %s\n", overlay_final, strerror(errno));
            ABORT(255);
        }

        singularity_message(VERBOSE, "Mounting overlay with options: %s\n", overlay_options);
        if ( mount("OverlayFS", overlay_final, "overlay", MS_NOSUID, overlay_options) < 0 ){
            singularity_message(ERROR, "Could not mount Singularity overlay: %s\n", strerror(errno));
            ABORT(255); 
        }
        singularity_priv_drop();

        free(overlay_mount);
        free(overlay_upper);
        free(overlay_options);

        singularity_registry_set("OVERLAYFS_ENABLED", "1");

        singularity_message(VERBOSE2, "Updating the containerdir to: %s\n", overlay_final);
        singularity_runtime_rootfs(overlay_final);

#else
        singularity_message(VERBOSE, "OverlayFS not supported by host build\n");
#endif
    }

    return(0);
}
