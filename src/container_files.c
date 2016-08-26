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


#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/types.h>
#include <limits.h>
#include <pwd.h>
#include <errno.h> 
#include <string.h>
#include <grp.h>


#include "config.h"
#include "file.h"
#include "util.h"
#include "message.h"
#include "privilege.h"


void update_passwd_file(char *file) {
    FILE *file_fp;
    uid_t uid = singularity_priv_getuid();
    errno = 0;
    struct passwd *pwent = getpwuid(uid);

    if ( !pwent ) {
        // List of potential error codes for unknown name taken from man page.
        if ( (errno == 0) || (errno == ESRCH) || (errno == EBADF) || (errno == EPERM) ) {
            message(VERBOSE3, "Not updating passwd file as entry for %d not found.\n", uid);
            return;
        } else {
            message(ERROR, "Failed to lookup username for UID %d: %s\n", uid, strerror(errno));
            ABORT(255);
        }
    }

    message(DEBUG, "Called update_passwd_file(%s)\n", file);

    message(VERBOSE2, "Checking for passwd file: %s\n", file);
    if ( is_file(file) < 0 ) {
        message(WARNING, "Template passwd not found: %s\n", file);
        return;
    }

    message(VERBOSE, "Updating passwd file with user info\n");
    if ( ( file_fp = fopen(file, "a") ) == NULL ) { // Flawfinder: ignore
        message(ERROR, "Could not open template passwd file %s: %s\n", file, strerror(errno));
        ABORT(255);
    }
    fprintf(file_fp, "\n%s:x:%d:%d:%s:%s:%s\n", pwent->pw_name, pwent->pw_uid, pwent->pw_gid, pwent->pw_gecos, pwent->pw_dir, pwent->pw_shell);
    fclose(file_fp);

}


void update_group_file(char *file) {
    FILE *file_fp;
    int i;
    uid_t uid = singularity_priv_getuid();
    uid_t gid = singularity_priv_getgid();
    const gid_t *gids = singularity_priv_getgids();
    int gid_count = singularity_priv_getgidcount();

    errno = 0;
    struct passwd *pwent = getpwuid(uid);
    if ( !pwent ) {
        // List of potential error codes for unknown name taken from man page.
        if ( (errno == 0) || (errno == ESRCH) || (errno == EBADF) || (errno == EPERM) ) {
            message(VERBOSE3, "Not updating group file as passwd entry for UID %d not found.\n", uid);
            return;
        } else {
            message(ERROR, "Failed to lookup username for UID %d: %s\n", uid, strerror(errno));
            ABORT(255);
        }
    }

    message(DEBUG, "Called update_group_file(%s)\n", file);

    message(VERBOSE2, "Checking for group file: %s\n", file);
    if ( is_file(file) < 0 ) {
        message(WARNING, "Template group file not found: %s\n", file);
        return;
    }
    if ( ( file_fp = fopen(file, "a") ) == NULL ) { // Flawfinder: ignore
        message(ERROR, "Could not open template group file %s: %s\n", file, strerror(errno));
        ABORT(255);
    }

    errno = 0;
    struct group *grent = getgrgid(gid);
    if ( grent ) {
        message(VERBOSE, "Updating group file with user info\n");
        fprintf(file_fp, "\n%s:x:%d:%s\n", grent->gr_name, grent->gr_gid, pwent->pw_name);
    } else if ( (errno == 0) || (errno == ESRCH) || (errno == EBADF) || (errno == EPERM) ) {
        // It's rare, but certainly possible to have a GID that's not a group entry in this system.
        // According to the man page, all of the above errno's can indicate this situation.
        message(VERBOSE3, "Skipping GID %d as group entry does not exist.\n", gid);
    } else {
        message(ERROR, "Failed to lookup GID %d group entry: %s\n", gid, strerror(errno));
        ABORT(255);
    }


    if ( !singularity_priv_userns_enabled() ) {
        message(DEBUG, "Getting supplementary group info\n");

        for (i=0; i < gid_count; i++) {
            errno = 0;
            struct group *gr = getgrgid(gids[i]);
            if ( gr ) {
                message(VERBOSE3, "Found supplementary group membership in: %d\n", gids[i]);
                if ( gids[i] != gid ) {
                    message(VERBOSE2, "Adding user's supplementary group ('%s') info to template group file\n", grent->gr_name);
                    fprintf(file_fp, "%s:x:%d:%s\n", gr->gr_name, gr->gr_gid, pwent->pw_name);
                }
            } else if ( (errno == 0) || (errno == ESRCH) || (errno == EBADF) || (errno == EPERM) ) {
                message(VERBOSE3, "Skipping GID %d as group entry does not exist.\n", gids[i]);
            } else {
                message(ERROR, "Failed to lookup GID %d group entry: %s\n", gids[i], strerror(errno));
                ABORT(255);
            }
        }
    }
    fclose(file_fp);

}
