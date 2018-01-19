/*
 * Copyright (c) 2016, Brian Bockelman. All rights reserved.
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

#define _GNU_SOURCE

#include <errno.h>
#include <fcntl.h>
#include <signal.h>
#include <string.h>
#include <unistd.h>
#include <poll.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <stdio.h>
#include <sys/file.h>
#include <sys/mount.h>

#include "util/message.h"
#include "util/util.h"
#include "util/file.h"
#include "util/registry.h"
#include "util/config_parser.h"
#include "util/fork.h"
#include "util/privilege.h"
#include "util/mount.h"

#ifndef LOCALSTATEDIR
#error LOCALSTATEDIR not defined
#endif


int singularity_sessiondir(void) {
    char *sessiondir = NULL;
    char *sessiondir_size_str = NULL;
    long int sessiondir_size = 0;
    int sessiondir_size_str_len;
    int sessiondir_size_str_usd;

    if ( singularity_registry_get("DAEMON_JOIN") ) {
        singularity_message(ERROR, "Internal Error - This function should not be called when joining an instance\n");
    }

    singularity_message(DEBUG, "Setting sessiondir\n");
    sessiondir = joinpath(LOCALSTATEDIR, "/singularity/mnt/session");
    singularity_message(VERBOSE, "Using session directory: %s\n", sessiondir);

    singularity_message(DEBUG, "Checking for session directory: %s\n", sessiondir);
    if ( is_dir(sessiondir) != 0 ) {
        singularity_message(ERROR, "Session directory does not exist: %s\n", sessiondir);
        ABORT(255);
    }

    singularity_message(DEBUG, "Obtaining the default sessiondir size\n");
    if ( str2int(singularity_config_get_value(SESSIONDIR_MAXSIZE), &sessiondir_size) < 0 ) {
        singularity_message(ERROR, "Failed converting sessiondir size to integer, check config file\n");
        ABORT(255);
    }
    singularity_message(DEBUG, "Converted sessiondir size to: %ld\n", sessiondir_size);

    singularity_message(DEBUG, "Creating the sessiondir size mount option length\n");
    sessiondir_size_str_len = intlen(sessiondir_size) + 7;

    singularity_message(DEBUG, "Got size length of: %d\n", sessiondir_size_str_len);
    sessiondir_size_str = (char *) malloc(sessiondir_size_str_len);

    singularity_message(DEBUG, "Creating the sessiondir size mount option string\n");
    sessiondir_size_str_usd = snprintf(sessiondir_size_str, sessiondir_size_str_len, "size=%ldm", sessiondir_size);

    singularity_message(DEBUG, "Checking to make sure the string was allocated correctly\n");
    if ( sessiondir_size_str_usd + 1 !=  sessiondir_size_str_len ) {
        singularity_message(ERROR, "Failed to allocate string for sessiondir size string (%s): %d + 1 != %d\n", sessiondir_size_str, sessiondir_size_str_usd, sessiondir_size_str_len);
        ABORT(255);
    }

    singularity_priv_escalate();
    singularity_message(DEBUG, "Mounting sessiondir tmpfs: %s\n", sessiondir);
    if ( singularity_mount("tmpfs", sessiondir, "tmpfs", MS_NOSUID, sessiondir_size_str) < 0 ){
        singularity_message(ERROR, "Failed to mount sessiondir tmpfs %s: %s\n", sessiondir, strerror(errno));
        ABORT(255);
    }
    singularity_priv_drop();

    singularity_registry_set("SESSIONDIR", sessiondir);

    return(0);
}

