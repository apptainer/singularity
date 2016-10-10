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
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/mount.h>
#include <unistd.h>
#include <stdlib.h>

#include "util/file.h"
#include "util/util.h"
#include "lib/message.h"
#include "lib/privilege.h"
#include "lib/ns/ns.h"


static char *source_dir = NULL;
static char *mount_point = NULL;
static int read_write = 0;


int rootfs_dir_init(char *source, char *mount_dir) {
    singularity_message(DEBUG, "Inializing container rootfs dir subsystem\n");

    if ( is_dir(source) < 0 ) {
        singularity_message(ERROR, "Container source directory is not available: %s\n", source);
        ABORT(255);
    }

    if ( strcmp(source, "/") == 0 ) {
        singularity_message(ERROR, "Naughty, naughty, naughty...!\n");
        ABORT(255);
    }

    source_dir = strdup(source);
    mount_point = strdup(mount_dir);

    if ( envar_defined("SINGULARITY_WRITABLE") == TRUE ) {
        read_write = 1;
    }

    return(0);
}


int rootfs_dir_mount(void) {

    if ( ( mount_point == NULL ) || ( source_dir == NULL ) ) {
        singularity_message(ERROR, "Called image_mount but image_init() hasn't been called\n");
        ABORT(255);
    }

    if ( is_dir(mount_point) < 0 ) {
        singularity_message(ERROR, "Container directory not available: %s\n", mount_point);
        ABORT(255);
    }

    singularity_priv_escalate();
    singularity_message(DEBUG, "Mounting container directory %s->%s\n", source_dir, mount_point);
    if ( mount(source_dir, mount_point, NULL, MS_BIND|MS_NOSUID|MS_REC, NULL) < 0 ) {
        singularity_message(ERROR, "Could not mount container directory %s->%s: %s\n", source_dir, mount_point, strerror(errno));
        return 1;
    }
    singularity_priv_drop();

    if ( read_write <= 0 ) {
        if ( singularity_ns_user_enabled() <= 0 ) {
            singularity_priv_escalate();
            singularity_message(VERBOSE2, "Making mount read only: %s\n", mount_point);
            if ( mount(NULL, mount_point, NULL, MS_BIND|MS_NOSUID|MS_REC|MS_REMOUNT|MS_RDONLY, NULL) < 0 ) {
                singularity_message(ERROR, "Could not bind read only %s: %s\n", mount_point, strerror(errno));
                ABORT(255);
            }
            singularity_priv_drop();
        }
    }

    return(0);
}

