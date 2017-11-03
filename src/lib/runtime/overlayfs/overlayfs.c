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

#include "config.h"
#include "util/file.h"
#include "util/util.h"
#include "util/registry.h"
#include "util/message.h"
#include "util/config_parser.h"
#include "util/privilege.h"
#include "util/mount.h"

#include "lib/image/image.h"

#include "../runtime.h"


int _singularity_runtime_overlayfs(void) {

    singularity_priv_escalate();
    singularity_message(DEBUG, "Creating overlay_final directory: %s\n", CONTAINER_FINALDIR);
    if ( s_mkpath(CONTAINER_FINALDIR, 0755) < 0 ) {
        singularity_message(ERROR, "Failed creating overlay_final directory %s: %s\n", CONTAINER_FINALDIR, strerror(errno));
        ABORT(255);
    }
    singularity_priv_drop();

    singularity_message(DEBUG, "Checking if overlayfs should be used\n");
    int try_overlay = ( strcmp("try", singularity_config_get_value(ENABLE_OVERLAY)) == 0 );
    if ( !try_overlay && ( singularity_config_get_bool_char(ENABLE_OVERLAY) <= 0 ) ) {
        singularity_message(VERBOSE3, "Not enabling overlayFS via configuration\n");
    } else if ( singularity_registry_get("DISABLE_OVERLAYFS") != NULL ) {
        singularity_message(VERBOSE3, "Not enabling overlayFS via environment\n");
    } else if ( singularity_registry_get("WRITABLE") != NULL ) {
        singularity_message(VERBOSE3, "Not enabling overlayFS, image mounted writable\n");
    } else {
        char *rootfs_source = CONTAINER_MOUNTDIR;
        char *overlay_final = CONTAINER_FINALDIR;
        char *overlay_mount = CONTAINER_OVERLAY;
        char *overlay_upper = joinpath(overlay_mount, "/upper");
        char *overlay_work  = joinpath(overlay_mount, "/work");
        int overlay_options_len = strlength(rootfs_source, PATH_MAX) + strlength(overlay_upper, PATH_MAX) + strlength(overlay_work, PATH_MAX) + 50;
        char *overlay_options = (char *) malloc(overlay_options_len);
        char *overlay_path = NULL;

        if (try_overlay)
            singularity_message(VERBOSE3, "Trying OverlayFS as requested by configuration\n");
        else
            singularity_message(VERBOSE3, "OverlayFS enabled by configuration\n");

        singularity_message(DEBUG, "Setting up overlay mount options\n");
        snprintf(overlay_options, overlay_options_len, "lowerdir=%s,upperdir=%s,workdir=%s", rootfs_source, overlay_upper, overlay_work); // Flawfinder: ignore

        singularity_message(DEBUG, "Checking for existance of overlay directory: %s\n", overlay_mount);
        if ( is_dir(overlay_mount) < 0 ) {
            singularity_message(ERROR, "Overlay mount directory does not exist: %s\n", overlay_mount);
            ABORT(255);
        }

        if ( ( overlay_path = singularity_registry_get("OVERLAYIMAGE") ) != NULL ) {
            struct image_object image;

            image = singularity_image_init(singularity_registry_get("OVERLAYIMAGE"), O_RDWR);

            if ( singularity_image_type(&image) != EXT3 ) {
                if ( singularity_image_type(&image) == DIRECTORY ) {
                    if ( singularity_priv_getuid() == 0 ) {
                        singularity_message(VERBOSE, "Allowing directory based overlay as root user\n");
                    } else {
                        singularity_message(ERROR, "Only root can use directory based overlays\n");
                        ABORT(255);
                    }
                } else {
                    singularity_message(ERROR, "Persistent overlay must be a writable image or directory\n");
                    ABORT(255);
                }
            }

            if ( singularity_image_mount(&image, overlay_mount) != 0 ) {
                singularity_message(ERROR, "Could not mount persistent overlay file: %s\n", singularity_image_name(&image));
                ABORT(255);
            }

        } else {
            char *size = NULL;

            if ( singularity_priv_getuid() == 0 ) {
                size = strdup("");
            } else {
                size = strdup("size=1m");
            }

            singularity_priv_escalate();
            singularity_message(DEBUG, "Mounting overlay tmpfs: %s\n", overlay_mount);
            if ( singularity_mount("tmpfs", overlay_mount, "tmpfs", MS_NOSUID | MS_NODEV, size) < 0 ){
                singularity_message(ERROR, "Failed to mount overlay tmpfs %s: %s\n", overlay_mount, strerror(errno));
                ABORT(255);
            }
            singularity_priv_drop();

            free(size);
        }

        if ( is_link(overlay_upper) == 0 ) {
            singularity_message(ERROR, "symlink detected, upper overlay %s must be a directory\n", overlay_upper);
            ABORT(255);
        }

        if ( is_link(overlay_work) == 0 ) {
            singularity_message(ERROR, "symlink detected, work overlay %s must be a directory\n", overlay_work);
            ABORT(255);
        }

        singularity_priv_escalate();
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

        singularity_message(VERBOSE, "Mounting overlay with options: %s\n", overlay_options);
        int result = singularity_mount("OverlayFS", overlay_final, "overlay", MS_NOSUID | MS_NODEV, overlay_options);
        if (result < 0) {
            if ( (errno == EPERM) || ( try_overlay && ( errno == ENODEV ) ) ) {
                singularity_message(VERBOSE, "Singularity overlay mount did not work (%s), continuing without it\n", strerror(errno));
                singularity_message(DEBUG, "Unmounting overlay tmpfs: %s\n", overlay_mount);
                umount(overlay_mount);
            } else {
                singularity_message(ERROR, "Could not mount Singularity overlay: %s\n", strerror(errno));
                ABORT(255); 
            }
        }
        singularity_priv_drop();

        free(overlay_upper);
        free(overlay_work);
        free(overlay_options);

        if (result >= 0) {
            singularity_registry_set("OVERLAYFS_ENABLED", "1");
            return(0);
        }
    }


    // If we got here, assume we are not overlaying, so we must bind to final directory
    singularity_priv_escalate();
    singularity_message(DEBUG, "Binding container directory to final home %s->%s\n", CONTAINER_MOUNTDIR, CONTAINER_FINALDIR);
    if ( singularity_mount(CONTAINER_MOUNTDIR, CONTAINER_FINALDIR, NULL, MS_BIND|MS_NOSUID|MS_REC|MS_NODEV, NULL) < 0 ) {
        singularity_message(ERROR, "Could not bind mount container to final home %s->%s: %s\n", CONTAINER_MOUNTDIR, CONTAINER_FINALDIR, strerror(errno));
        return 1;
    }
    singularity_priv_drop();

    return(0);
}
