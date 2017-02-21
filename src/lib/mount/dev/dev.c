/* 
 * Copyright (c) 2015-2016, Gregory M. Kurtzer. All rights reserved.
 * 
 * “Singularity” Copyright (c) 2016, The Regents of the University of California,
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

#include "util/file.h"
#include "util/util.h"
#include "lib/message.h"
#include "lib/privilege.h"
#include "lib/config_parser.h"
#include "lib/ns/ns.h"
#include "lib/rootfs/rootfs.h"

static int mount_dev(const char *dev) {
    char *container_dir = singularity_rootfs_dir();
    char *path = joinpath(container_dir, dev);

    singularity_message(DEBUG, "Mounting device %s at %s\n", dev, path);

    if ( is_chr(path) == 0 ) {
        return(0);
    }

    if ( fileput(path, "") < 0 ) {
        singularity_message(VERBOSE, "Can not create %s: %s\n", dev, strerror(errno));
        return(-1);
    }

    if ( mount(dev, path, NULL, MS_BIND, NULL) < 0 ) {
        unlink(path);
        singularity_message(VERBOSE, "Can not mount %s: %s\n", dev, strerror(errno));
    }

    return(0);
}

int singularity_mount_dev(void) {
    char *container_dir = singularity_rootfs_dir();

    if ( strcmp("minimal", singularity_config_get_value(MOUNT_DEV)) == 0 ) {
        if ( singularity_rootfs_overlay_enabled() > 0 ) {
            if ( is_dir(joinpath(container_dir, "/dev")) < 0 ) {
                if ( s_mkpath(joinpath(container_dir, "/dev"), 0755) < 0 ) {
                    singularity_message(VERBOSE2, "Could not create /dev inside container, returning...\n");
                    return(0);
                }
            }

            singularity_priv_escalate();

            mount_dev("/dev/null");
            mount_dev("/dev/zero");
            mount_dev("/dev/random");
            mount_dev("/dev/urandom");

            if ( is_dir(joinpath(container_dir, "/dev/shm")) < 0 ) {
                if ( s_mkpath(joinpath(container_dir, "/dev/shm"), 0755) < 0 ) {
                    singularity_message(VERBOSE2, "Could not create /dev/shm inside container, returning...\n");
                    return(0);
                }
            }

            if ( mount("/dev/shm", joinpath(container_dir, "/dev/shm"), NULL, MS_BIND, NULL) < 0 ) {
                unlink(joinpath(container_dir, "/dev/shm"));
                singularity_message(VERBOSE, "Can not mount /dev/shm: %s\n", strerror(errno));
            }

            singularity_priv_drop();

            return(0);
        } else {
            singularity_message(ERROR, "Config option 'mount dev = minimal', requires overlayFS\n");
            ABORT(255);
        }
    }

    singularity_message(DEBUG, "Checking configuration file for 'mount dev'\n");
    if ( singularity_config_get_bool_char(MOUNT_DEV) > 0 ) {
        if ( is_dir(joinpath(container_dir, "/dev")) == 0 ) {
                singularity_priv_escalate();
                singularity_message(VERBOSE, "Bind mounting /dev\n");
                if ( mount("/dev", joinpath(container_dir, "/dev"), NULL, MS_BIND|MS_NOSUID|MS_REC, NULL) < 0 ) {
                    singularity_message(ERROR, "Could not bind mount container's /dev: %s\n", strerror(errno));
                    ABORT(255);
                }
                if ( singularity_priv_userns_enabled() != 1 ) {
                    if ( mount(NULL, joinpath(container_dir, "/dev"), NULL, MS_BIND|MS_NOSUID|MS_REC|MS_REMOUNT, NULL) < 0 ) {
                        singularity_message(ERROR, "Could not remount container's /dev: %s\n", strerror(errno));
                        ABORT(255);
                    }
                }
                singularity_priv_drop();
        } else {
            singularity_message(WARNING, "Not mounting /dev, container has no bind directory\n");
        }
        return(0);
    }

    singularity_message(VERBOSE, "Not mounting /dev inside the container\n");

    return(0);
}
