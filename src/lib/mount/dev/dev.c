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


int singularity_mount_dev(void) {
    char *container_dir = singularity_rootfs_dir();

    singularity_config_rewind();
    if ( strcmp("minimal", singularity_config_get_value("mount dev")) == 0 ) {
        if ( singularity_rootfs_overlay_enabled() > 0 ) {
            if ( is_dir(joinpath(container_dir, "/dev")) < 0 ) {
                if ( s_mkpath(joinpath(container_dir, "/dev"), 0755) < 0 ) {
                    singularity_message(VERBOSE2, "Could not create /dev inside container, returning...\n");
                    return(0);
                }
            }

            singularity_priv_escalate();

            singularity_message(DEBUG, "Checking container's /dev/null\n");
            if ( is_chr(joinpath(container_dir, "/dev/null")) < 0 ) {
                if ( mknod(joinpath(container_dir, "/dev/null"), S_IFCHR | 0666, makedev(1, 3)) < 0 ) {
                    singularity_message(VERBOSE, "Can not create /dev/null: %s\n", strerror(errno));
                }
            }

            singularity_message(DEBUG, "Checking container's /dev/zero\n");
            if ( is_chr(joinpath(container_dir, "/dev/zero")) < 0 ) {
                if ( mknod(joinpath(container_dir, "/dev/zero"), S_IFCHR | 0644, makedev(1, 5)) < 0 ) {
                    singularity_message(VERBOSE, "Can not create /dev/null: %s\n", strerror(errno));
                }
            }

            singularity_message(DEBUG, "Checking container's /dev/random\n");
            if ( is_chr(joinpath(container_dir, "/dev/random")) < 0 ) {
                if ( mknod(joinpath(container_dir, "/dev/random"), S_IFCHR | 0644, makedev(1, 8)) < 0 ) {
                    singularity_message(VERBOSE, "Can not create /dev/random: %s\n", strerror(errno));
                }
            }

            singularity_message(DEBUG, "Checking container's /dev/urandom\n");
            if ( is_chr(joinpath(container_dir, "/dev/urandom")) < 0 ) {
                if ( mknod(joinpath(container_dir, "/dev/urandom"), S_IFCHR | 0644, makedev(1, 9)) < 0 ) {
                    singularity_message(VERBOSE, "Can not create /dev/urandom: %s\n", strerror(errno));
                }
            }

            singularity_priv_drop();

            return(0);
        } else {
            singularity_message(VERBOSE2, "Not enabling 'mount dev = minimal', overlayfs not enabled\n");
        }
    }

    singularity_message(DEBUG, "Checking configuration file for 'mount dev'\n");
    singularity_config_rewind();
    if ( singularity_config_get_bool("mount dev", 1) > 0 ) {
        if ( is_dir(joinpath(container_dir, "/dev")) == 0 ) {
                singularity_priv_escalate();
                singularity_message(VERBOSE, "Bind mounting /dev\n");
                if ( mount("/dev", joinpath(container_dir, "/dev"), NULL, MS_BIND|MS_NOSUID|MS_REC, NULL) < 0 ) {
                    singularity_message(ERROR, "Could not bind mount container's /dev: %s\n", strerror(errno));
                    ABORT(255);
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
