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
#include <limits.h>
#include <unistd.h>
#include <stdlib.h>
#include <grp.h>
#include <pwd.h>


#include "file.h"
#include "util.h"
#include "message.h"
#include "privilege.h"
#include "sessiondir.h"
#include "rootfs/rootfs.h"
#include "file/file.h"


int singularity_file_group(void) {
    FILE *file_fp;
    char *source_file;
    char *tmp_file;
    int i;
    uid_t uid = priv_getuid();
    uid_t gid = priv_getgid();
    const gid_t *gids = priv_getgids();
    int gid_count = priv_getgidcount();
    struct passwd *pwent = getpwuid(uid);
    struct group *grent = getgrgid(gid);
    char *containerdir = singularity_rootfs_dir();
    char *sessiondir = singularity_sessiondir_get();

    message(DEBUG, "Called singularity_file_group_create()\n");

    if ( uid == 0 ) {
        message(VERBOSE, "Not updating group file, running as root!\n");
        return(0);
    }

    if ( containerdir == NULL ) {
        message(ERROR, "Failed to obtain container directory\n");
        ABORT(255);
    }

    if ( sessiondir == NULL ) {
        message(ERROR, "Failed to obtain session directory\n");
        ABORT(255);
    }

    source_file = joinpath(containerdir, "/etc/group");
    tmp_file = joinpath(sessiondir, "/group");

    if ( is_file(source_file) < 0 ) {
        message(VERBOSE, "Group file does not exist in container, not updating\n");
        return(0);
    }

    errno = 0;
    if ( ! pwent ) {
        // List of potential error codes for unknown name taken from man page.
        if ( (errno == 0) || (errno == ESRCH) || (errno == EBADF) || (errno == EPERM) ) {
            message(VERBOSE3, "Not updating group file as passwd entry for UID %d not found.\n", uid);
            return(0);
        } else {
            message(ERROR, "Failed to lookup username for UID %d: %s\n", uid, strerror(errno));
            ABORT(255);
        }
    }

    message(VERBOSE2, "Creating template of /etc/group for containment\n");
    if ( ( copy_file(source_file, tmp_file) ) < 0 ) {
        message(ERROR, "Failed copying template group file to sessiondir: %s\n", strerror(errno));
        ABORT(255);
    }

    if ( ( file_fp = fopen(tmp_file, "a") ) == NULL ) { // Flawfinder: ignore
        message(ERROR, "Could not open template group file %s: %s\n", tmp_file, strerror(errno));
        ABORT(255);
    }

    errno = 0;
    if ( grent ) {
        message(VERBOSE, "Updating group file with user info\n");
        fprintf(file_fp, "\n%s:x:%u:%s\n", grent->gr_name, grent->gr_gid, pwent->pw_name);
    } else if ( (errno == 0) || (errno == ESRCH) || (errno == EBADF) || (errno == EPERM) ) {
        // It's rare, but certainly possible to have a GID that's not a group entry in this system.
        // According to the man page, all of the above errno's can indicate this situation.
        message(VERBOSE3, "Skipping GID %d as group entry does not exist.\n", gid);
    } else {
        message(ERROR, "Failed to lookup GID %d group entry: %s\n", gid, strerror(errno));
        ABORT(255);
    }


    message(DEBUG, "Getting supplementary group info\n");

    for (i=0; i < gid_count; i++) {
        if ( gids[i] == gid ) {
            message(DEBUG, "Skipping duplicate supplementary group\n");
            continue;
        }

        if ( gids[i] < UINT_MAX && gids[i] >= 500 ) {
            errno = 0;
            struct group *gr = getgrgid(gids[i]);
            if ( gr ) {
                message(VERBOSE3, "Found supplementary group membership in: %d\n", gids[i]);
                message(VERBOSE2, "Adding user's supplementary group ('%s') info to template group file\n", grent->gr_name);
                fprintf(file_fp, "%s:x:%u:%s\n", gr->gr_name, gr->gr_gid, pwent->pw_name);
            } else if ( (errno == 0) || (errno == ESRCH) || (errno == EBADF) || (errno == EPERM) ) {
                message(VERBOSE3, "Skipping GID %d as group entry does not exist.\n", gids[i]);
            } else {
                message(ERROR, "Failed to lookup GID %d group entry: %s\n", gids[i], strerror(errno));
                ABORT(255);
            }
        } else {
            message(VERBOSE, "Group id '%d' is out of bounds\n", gids[i]);
        }
    }

    fclose(file_fp);


    container_file_bind("group", "/etc/group");

    return(0);
}
