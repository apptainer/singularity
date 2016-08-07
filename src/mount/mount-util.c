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
#include <unistd.h>
#include <stdlib.h>

#include "file.h"
#include "util.h"
#include "message.h"
#include "privilege.h"



void mount_bind(char * source, char * dest, int writable, const char *tmp_dir) {

    message(DEBUG, "Called mount_bind(%s, %s, %d, %s)\n", source, dest, writable, tmp_dir);

    message(DEBUG, "Checking that source exists and is a file or directory\n");
    if ( is_dir(source) != 0 && is_file(source) != 0 ) {
        message(ERROR, "Bind source path is not a file or directory: '%s'\n", source);
        ABORT(255);
    }

    message(DEBUG, "Checking that destination exists and is a file or directory\n");
    if ( is_dir(dest) != 0 && is_file(dest) != 0 ) {
        if ( create_bind_dir(dest, tmp_dir, is_dir(source)) != 0 ) {
            message(ERROR, "Container bind path is not a file or directory: '%s'\n", dest);
            ABORT(255);
        }
    }

    //  NOTE: The kernel history is a bit ... murky ... as to whether MS_RDONLY can be set in the
    //  same syscall as the bind.  It seems to no longer work on modern kernels - hence, we also
    //  do it below.  *However*, if we are using user namespaces, we get an EPERM error on the
    //  separate mount command below.  Hence, we keep the flag in the first call until the kernel
    //  picture cleras up.
    message(DEBUG, "Calling mount(%s, %s, ...)\n", source, dest);
    if ( mount(source, dest, NULL, MS_BIND|MS_NOSUID|MS_REC|(writable <= 0 ? MS_RDONLY : 0), NULL) < 0 ) {
        message(ERROR, "Could not bind %s: %s\n", dest, strerror(errno));
        ABORT(255);
    }

    // Note that we can't remount as read-only if we are in unprivileged mode.
    if ( !priv_userns_enabled() && (writable <= 0) ) {
        message(VERBOSE2, "Making mount read only: %s\n", dest);
        if ( mount(NULL, dest, NULL, MS_BIND|MS_REC|MS_REMOUNT|MS_RDONLY, NULL) < 0 ) {
            message(ERROR, "Could not bind read only %s: %s\n", dest, strerror(errno));
            ABORT(255);
        }
    }
    message(DEBUG, "Returning mount_bind(%s, %d, %d) = 0\n", source, dest, writable);
}
