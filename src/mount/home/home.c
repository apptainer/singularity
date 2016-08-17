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
#include "sessiondir.h"
#include "rootfs/rootfs.h"


int singularity_mount_home(void) {
    char *homedir;
    char *homedir_base;
    char *container_dir = singularity_rootfs_dir();
    struct passwd *pw;
    uid_t uid = priv_getuid();

    config_rewind();
    if ( config_get_key_bool("mount home", 1) <= 0 ) {
        message(VERBOSE, "Skipping tmp dir mounting (per config)\n");
        return(0);
    }

    errno = 0;
    if ( ( pw = getpwuid(uid) ) == NULL ) {
        // List of potential error codes for unknown name taken from man page.
        if ( (errno == 0) || (errno == ESRCH) || (errno == EBADF) || (errno == EPERM) ) {
            message(VERBOSE3, "Not mounting home directory as passwd entry for %d not found.\n", uid);
            return(1);
        } else {
            message(ERROR, "Failed to lookup username for UID %d: %s\n", getuid, strerror(errno));
            ABORT(255);
        }
    }

    message(DEBUG, "Obtaining user's homedir\n");
    homedir = pw->pw_dir;

    //TODO: Find out if we can create mount points here...

    if ( ( homedir_base = container_basedir(container_dir, homedir) ) != NULL ) {
        char *homedir_base_source;

        if ( getenv("SINGULARITY_CONTAIN") != NULL ) {
            char *sessiondir = singularity_sessiondir_get();

            homedir_base_source = joinpath(sessiondir, homedir_base);

            s_mkpath(joinpath(sessiondir, homedir), 0750);
        } else {
            homedir_base_source = strdup(homedir_base);
        }

        if ( is_dir(homedir_base_source) == 0 ) {
            if ( is_dir(joinpath(container_dir, homedir_base)) == 0 ) {
                priv_escalate();
                message(VERBOSE, "Mounting home directory base path: %s\n", homedir_base);
                if ( mount(homedir_base_source, joinpath(container_dir, homedir_base), NULL, MS_BIND|MS_NOSUID|MS_REC, NULL) < 0 ) {
                    message(ERROR, "Failed to mount home directory: %s\n", strerror(errno));
                    ABORT(255);
                }
                priv_drop();
            } else {
                message(WARNING, "Container bind point does not exist: '%s' (homedir_base)\n", homedir_base);
            }
        } else {
            message(WARNING, "Home directory base source path does not exist: %s\n", homedir_base);
        }
    }
    return(0);
}
