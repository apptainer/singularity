/*
 * Copyright (c) 2017-2018, SyLabs, Inc. All rights reserved.
 * Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
 *
 * This software is licensed under a 3-clause BSD license.  Please
 * consult LICENSE file distributed with the sources of this project regarding
 * your rights to use or distribute this software.
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

#include "util/file.h"
#include "util/util.h"
#include "util/registry.h"
#include "util/config_parser.h"
#include "util/message.h"
#include "util/privilege.h"

#include "runtime/file_bind.h"
#include "runtime/runtime.h"


int _singularity_runtime_files_hostname(void) {
    FILE *hostname_fd;
    char *tmpdir = singularity_registry_get("SESSIONDIR");
    char *hostname_file = "/etc/hostname";
    char *containerdir = CONTAINER_FINALDIR;
    char *source = joinpath(tmpdir, "/hostname");
    char *target = joinpath(containerdir, hostname_file);
    char *hostname = singularity_registry_get("HOSTNAME");

    if ( hostname == NULL ) {
        singularity_message(DEBUG, "Setting container hostname not requested by user\n");
        return(0);
    }

    singularity_message(DEBUG, "Check if /etc/hostname is present in container\n");
    if ( is_file(target) < 0 ) {
        singularity_message(VERBOSE, "/etc/hostname doesn't exists, skipping\n");
        return(0);
    }

    hostname_fd = fopen(source, "w+");
    if ( hostname_fd == NULL ) {
        singularity_message(ERROR, "Couldn't create hostname session file\n");
        ABORT(255);
    }

    if ( strlen(hostname) > HOST_NAME_MAX ) {
        hostname[HOST_NAME_MAX] = '\0';
    }

    fprintf(hostname_fd, "%s\n", hostname);
    fclose(hostname_fd);

    container_file_bind(source, hostname_file);

    return(0);
}
