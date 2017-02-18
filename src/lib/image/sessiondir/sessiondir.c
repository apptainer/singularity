/* 
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
#include <sys/file.h>
#include <sys/wait.h>
#include <unistd.h>
#include <stdlib.h>
#include <limits.h>

#include "util/file.h"
#include "util/util.h"
#include "util/registry.h"
#include "util/message.h"
#include "util/privilege.h"
#include "util/config_parser.h"
#include "util/fork.h"

#include "../image.h"
#include "./sessiondir.h"


//void singularity_image_sessiondir_init(struct image_object *image);
//int singularity_image_sessiondir_create(struct image_object *image);
//int singularity_image_sessiondir_remove(struct image_object *image);



void _singularity_image_sessiondir_init(struct image_object *image) {
    char *sessiondir_prefix;
    char *sessiondir_suffix;
    char *file = strdup(image->path);
    struct stat imagestat;
    int sessiondir_suffix_len;
    uid_t uid = singularity_priv_getuid();
    int image_fd = image->fd;

    if ( image->sessiondir != NULL ) {
        singularity_message(DEBUG, "Called singularity_image_sessiondir_init previously, returning\n");
        return;
    }

    if ( ( sessiondir_prefix = singularity_registry_get("SESSIONDIR") ) != NULL ) {
        singularity_message(DEBUG, "Got sessiondir_prefix from environment: '%s'\n", sessiondir_prefix);
    } else if ( ( sessiondir_prefix = strdup(singularity_config_get_value(SESSIONDIR_PREFIX)) ) != NULL ) {
        singularity_message(DEBUG, "Got sessiondir_prefix from configuration: '%s'\n", sessiondir_prefix);
    } else {
        singularity_message(ERROR, "Could not obtain the session directory prefix.\n");
        ABORT(255);
    }
    singularity_message(DEBUG, "Set sessiondir_prefix to: %s\n", sessiondir_prefix);

    if ( fstat(image_fd, &imagestat) < 0 ) {
        singularity_message(ERROR, "Failed calling stat() on %s: %s\n", file, strerror(errno));
        ABORT(255);
    }

    sessiondir_suffix_len = intlen((int)uid) + intlen((int)imagestat.st_dev) + intlen((long unsigned)imagestat.st_ino) + 3;

    sessiondir_suffix = (char *) malloc(sessiondir_suffix_len);

    singularity_message(DEBUG, "Setting sessiondir suffix to: '%d.%d.%lu'\n", (int)uid, (int)imagestat.st_dev, (long unsigned)imagestat.st_ino);

    if ( snprintf(sessiondir_suffix, sessiondir_suffix_len, "%d.%d.%lu", (int)uid, (int)imagestat.st_dev, (long unsigned)imagestat.st_ino) < 0 ) {
        singularity_message(ERROR, "Failed creating sessiondir_suffix: %s\n", sessiondir_suffix);
        ABORT(255);
    }

    if ( ( image->sessiondir = strcat(sessiondir_prefix, sessiondir_suffix) ) == NULL ) {
        singularity_message(ERROR, "Could not set image->sessiondir\n");
        ABORT(255);
    }

    singularity_registry_set("sessiondir", image->sessiondir);

    singularity_message(VERBOSE, "Creating session directory: %s\n", image->sessiondir);

    if ( s_mkpath(image->sessiondir, 0755) < 0 ) {
        singularity_message(ERROR, "Failed creating session directory %s: %s\n", image->sessiondir, strerror(errno));
        ABORT(255);
    }

    singularity_message(DEBUG, "Opening sessiondir file descriptor\n");
    if ( ( image->sessiondir_fd = open(image->sessiondir, O_RDONLY) ) < 0 ) { // Flawfinder: ignore
        singularity_message(ERROR, "Could not obtain file descriptor for session directory %s: %s\n", image->sessiondir, strerror(errno));
        ABORT(255);
    }

    singularity_message(DEBUG, "Setting shared flock() on session directory\n");
    if ( flock(image->sessiondir_fd, LOCK_SH | LOCK_NB) < 0 ) {
        singularity_message(ERROR, "Could not obtain shared lock on %s: %s\n", image->sessiondir, strerror(errno));
        ABORT(255);
    }

    if ( singularity_registry_get("NOSESSIONCLEANUP") == NULL ) {
        int child = singularity_fork();

        if ( child == 0 ) {
            char *cleanup_proc[2];

            cleanup_proc[0] = joinpath(LIBEXECDIR, "/singularity/bin/cleanup");
            cleanup_proc[1] = NULL;

            setenv("SINGULARITY_CLEANDIR", image->sessiondir, 1);
            close(image->sessiondir_fd);

            execv(cleanup_proc[0], cleanup_proc);

        } else if ( child > 0 ) {
            int tmpstatus;

            waitpid(child, &tmpstatus, 0);
            if ( WEXITSTATUS(tmpstatus) != 0 ) {
                singularity_message(ERROR, "Failed to spawn cleanup daemon process\n");
                ABORT(255);
            }
        }
    }

    return;
}
