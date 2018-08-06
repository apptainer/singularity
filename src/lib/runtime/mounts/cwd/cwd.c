/* 
 * Copyright (c) 2017-2018, SyLabs, Inc. All rights reserved.
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

#ifndef _GNU_SOURCE
#define _GNU_SOURCE
#endif

#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <string.h>
#include <sys/mount.h>
#include <sys/stat.h>
#include <unistd.h>
#include <stdlib.h>
#include <libgen.h>
#include <linux/limits.h>

#include "config.h"
#include "util/file.h"
#include "util/util.h"
#include "util/message.h"
#include "util/config_parser.h"
#include "util/registry.h"
#include "util/mountlist.h"

#include "../../runtime.h"


int _singularity_runtime_mount_cwd(struct mountlist *mountlist) {
    char *cwd_path = (char *)malloc(PATH_MAX);

    singularity_message(DEBUG, "Checking to see if we should mount current working directory\n");
    if ( cwd_path == NULL ) {
        singularity_message(ERROR, "Could not allocate memory for current working directory\n");
        ABORT(255);
    }

    singularity_message(DEBUG, "Getting current working directory\n");
    cwd_path[PATH_MAX-1] = '\0';
    if ( getcwd(cwd_path, PATH_MAX-1) == NULL ) {
        singularity_message(ERROR, "Could not obtain current directory path: %s\n", strerror(errno));
        ABORT(1);
    }

    singularity_message(DEBUG, "Checking for contain option\n");
    if ( singularity_registry_get("CONTAIN") != NULL ) {
        singularity_message(VERBOSE, "Not mounting current directory: contain was requested\n");
        free(cwd_path);
        return(0);
    }

    // Note that the cwd_path could be under a directory already on the mount
    // list, in which case the bind mount won't help, but it won't hurt either.

    singularity_message(DEBUG, "Checking if cwd is in an operating system directory\n");
    if ( ( strcmp(cwd_path, "/") == 0 ) ||
         ( strcmp(cwd_path, "/bin") == 0 ) ||
         ( strcmp(cwd_path, "/etc") == 0 ) ||
         ( strcmp(cwd_path, "/mnt") == 0 ) ||
         ( strcmp(cwd_path, "/usr") == 0 ) ||
         ( strcmp(cwd_path, "/var") == 0 ) ||
         ( strcmp(cwd_path, "/opt") == 0 ) ||
         ( strcmp(cwd_path, "/sbin") == 0 ) ) {
        singularity_message(VERBOSE, "Not mounting CWD within operating system directory: %s\n", cwd_path);
        free(cwd_path);
        return(0);
    }

    singularity_message(DEBUG, "Checking if cwd is in a virtual directory\n");
    if ( ( strncmp(cwd_path, "/sys", 4) == 0 ) ||
         ( strncmp(cwd_path, "/dev", 4) == 0 ) ||
         ( strncmp(cwd_path, "/proc", 5) == 0 ) ) {
        singularity_message(VERBOSE, "Not mounting CWD within virtual directory: %s\n", cwd_path);
        free(cwd_path);
        return(0);
    }

    singularity_message(DEBUG, "Checking configuration file for 'user bind control'\n");
    if ( singularity_config_get_bool(USER_BIND_CONTROL) <= 0 ) {
        singularity_message(WARNING, "Not mounting current directory: user bind control is disabled by system administrator\n");
        free(cwd_path);
        return(0);
    }

    singularity_message(VERBOSE, "Queuing bind mount of '%s' to '%s' if mountpoint exists\n", cwd_path, cwd_path);
    mountlist_add(mountlist, NULL, cwd_path, NULL, MS_BIND|MS_NOSUID|MS_NODEV|MS_REC, ML_ONLY_IF_POINT_PRESENT);

    return(0);
}

