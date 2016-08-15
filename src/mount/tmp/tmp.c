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
#include <pwd.h>

#include "file.h"
#include "util.h"
#include "message.h"
#include "privilege.h"
#include "config_parser.h"
#include "rootfs/rootfs.h"


int singularity_mount_tmp(void) {
    char *container_dir = singularity_rootfs_dir();

    if ( getenv("SINGULARITY_CONTAIN") != NULL ) {
        message(DEBUG, "Skipping bind mounts as contain was requested\n");
        return(0);
    }

    config_rewind();
    if ( config_get_key_bool("mount tmp", 1) <= 0 ) {
        message(VERBOSE, "Skipping tmp dir mounting (per config)\n");
        return(0);
    }

    if ( is_dir("/tmp") == 0 ) {
        if ( is_dir(joinpath(container_dir, "/tmp")) == 0 ) {
            priv_escalate();
            message(VERBOSE, "Mounting directory: /tmp\n");
            if ( mount("/tmp", joinpath(container_dir, "/tmp"), NULL, MS_BIND|MS_NOSUID|MS_REC, NULL) < 0 ) {
                message(ERROR, "Failed to mount /tmp: %s\n", strerror(errno));
                ABORT(255);
            }
            priv_drop();
        } else {
            message(VERBOSE, "Could not mount container's /tmp directory: does not exist\n");
        }
    } else {
        message(VERBOSE, "Could not mount host's /tmp directory: does not exist\n");
    }

    if ( is_dir("/var/tmp") == 0 ) {
        if ( is_dir(joinpath(container_dir, "/var/tmp")) == 0 ) {
            priv_escalate();
            message(VERBOSE, "Mounting directory: /var/tmp\n");
            if ( mount("/var/tmp", joinpath(container_dir, "/var/tmp"), NULL, MS_BIND|MS_NOSUID|MS_REC, NULL) < 0 ) {
                message(ERROR, "Failed to mount /var/tmp: %s\n", strerror(errno));
                ABORT(255);
            }
            priv_drop();
        } else {
            message(VERBOSE, "Could not mount container's /var/tmp directory: does not exist\n");
        }
    } else {
        message(VERBOSE, "Could not mount host's /var/tmp directory: does not exist\n");
    }


    return(0);
}
