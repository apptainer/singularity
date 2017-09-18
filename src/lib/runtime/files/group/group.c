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
#include <limits.h>
#include <unistd.h>
#include <stdlib.h>
#include <grp.h>
#include <pwd.h>

#include "config.h"
#include "util/file.h"
#include "util/util.h"
#include "util/config_parser.h"
#include "util/message.h"
#include "util/privilege.h"
#include "util/registry.h"

#include "../file-bind.h"
#include "../../runtime.h"


int _singularity_runtime_files_group(void) {
    FILE *file_fp;
    char *source_file;
    char *tmp_file;
    int i;
    uid_t uid = singularity_priv_getuid();
    uid_t gid = singularity_priv_getgid();
    const gid_t *gids = singularity_priv_getgids();
    int gid_count = singularity_priv_getgidcount();
    char *containerdir = CONTAINER_FINALDIR;
    char *tmpdir = singularity_registry_get("SESSIONDIR");

    singularity_message(DEBUG, "Called singularity_file_group_create()\n");

    if ( uid == 0 ) {
        singularity_message(VERBOSE, "Not updating group file, running as root!\n");
        return(0);
    }

    if ( containerdir == NULL ) {
        singularity_message(ERROR, "Failed to obtain container directory\n");
        ABORT(255);
    }

    if ( tmpdir == NULL ) {
        singularity_message(ERROR, "Failed to obtain session directory\n");
        ABORT(255);
    }

    singularity_message(DEBUG, "Checking configuration option: 'config group'\n");
    if ( singularity_config_get_bool(CONFIG_GROUP) <= 0 ) {
        singularity_message(VERBOSE, "Skipping bind of the host's /etc/group\n");
        return(0);
    }

    source_file = joinpath(containerdir, "/etc/group");
    tmp_file = joinpath(tmpdir, "/group");

    if ( is_file(source_file) < 0 ) {
        singularity_message(VERBOSE, "Group file does not exist in container, not updating\n");
        return(0);
    }

    errno = 0;
    struct passwd *pwent = getpwuid(uid);
    if ( ! pwent ) {
        // List of potential error codes for unknown name taken from man page.
        if ( (errno == 0) || (errno == ESRCH) || (errno == EBADF) || (errno == EPERM) || (errno == ENOENT)) {
            singularity_message(VERBOSE3, "Not updating group file as passwd entry for UID %d not found.\n", uid);
            return(0);
        } else {
            singularity_message(ERROR, "Failed to lookup username for UID %d: %s\n", uid, strerror(errno));
            ABORT(255);
        }
    }

    singularity_message(VERBOSE2, "Creating template of /etc/group for containment\n");
    if ( ( copy_file(source_file, tmp_file) ) < 0 ) {
        singularity_message(ERROR, "Failed copying template group file to tmpdir: %s\n", strerror(errno));
        ABORT(255);
    }

    if ( ( file_fp = fopen(tmp_file, "a") ) == NULL ) { // Flawfinder: ignore
        singularity_message(ERROR, "Could not open template group file %s: %s\n", tmp_file, strerror(errno));
        ABORT(255);
    }

    errno = 0;
    struct group *grent = getgrgid(gid);
    if ( grent ) {
        singularity_message(VERBOSE, "Updating group file with user info\n");
        fprintf(file_fp, "\n%s:x:%u:%s\n", grent->gr_name, grent->gr_gid, pwent->pw_name);
    } else if ( (errno == 0) || (errno == ESRCH) || (errno == EBADF) || (errno == EPERM) || (errno == ENOENT) ) {
        // It's rare, but certainly possible to have a GID that's not a group entry in this system.
        // According to the man page, all of the above errno's can indicate this situation.
        singularity_message(VERBOSE3, "Skipping GID %d as group entry does not exist.\n", gid);
    } else {
        singularity_message(ERROR, "Failed to lookup GID %d group entry: %s\n", gid, strerror(errno));
        ABORT(255);
    }


    singularity_message(DEBUG, "Getting supplementary group info\n");

    for (i=0; i < gid_count; i++) {
        if ( gids[i] == gid ) {
            singularity_message(DEBUG, "Skipping duplicate supplementary group\n");
            continue;
        }

        if ( gids[i] < UINT_MAX ) {
            errno = 0;
            struct group *gr = getgrgid(gids[i]);
            if ( gr ) {
                singularity_message(VERBOSE3, "Found supplementary group membership in: %d\n", gids[i]);
                singularity_message(VERBOSE2, "Adding user's supplementary group ('%s') info to template group file\n", gr->gr_name);
                fprintf(file_fp, "%s:x:%u:%s\n", gr->gr_name, gr->gr_gid, pwent->pw_name);
            } else if ( (errno == 0) || (errno == ESRCH) || (errno == EBADF) || (errno == EPERM) ) {
                singularity_message(VERBOSE3, "Skipping GID %d as group entry does not exist.\n", gids[i]);
            } else {
                singularity_message(ERROR, "Failed to lookup GID %d group entry: %s\n", gids[i], strerror(errno));
                ABORT(255);
            }
        } else {
            singularity_message(VERBOSE, "Group id '%d' is out of bounds\n", gids[i]);
        }
    }

    fclose(file_fp);


    container_file_bind(tmp_file, "/etc/group");

    return(0);
}
