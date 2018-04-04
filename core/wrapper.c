/*
 * Copyright (c) 2017-2018, SyLabs, Inc. All rights reserved.
 * Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
 *
 * This software is licensed under a 3-clause BSD license.  Please
 * consult LICENSE file distributed with the sources of this project regarding
 * your rights to use or distribute this software.
 *
 */


#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <errno.h>
#include <string.h>
#include <fcntl.h>
#include <sys/mount.h>

#include "lib/util/file.h"
#include "lib/util/util.h"
#include "lib/util/registry.h"
#include "lib/util/config_parser.h"
#include "lib/util/capability.h"
#include "lib/util/privilege.h"
#include "lib/util/suid.h"
#include "lib/image/image.h"
#include "lib/runtime/runtime.h"

#ifndef SYSCONFDIR
#error SYSCONFDIR not defined
#endif

#define MOUNT_BINARY    "mount"
#define START_BINARY    "start"
#define ACTION_BINARY   "action"

extern char **environ;

struct cmd_wrapper {
    char *command;
    char *binary;
    void (*capinit)(void);
};

struct cmd_wrapper cmd_wrapper[] = {
    { .command = "shell",           .binary = ACTION_BINARY, .capinit = singularity_capability_init },
    { .command = "exec",            .binary = ACTION_BINARY, .capinit = singularity_capability_init },
    { .command = "run",             .binary = ACTION_BINARY, .capinit = singularity_capability_init },
    { .command = "test",            .binary = ACTION_BINARY, .capinit = singularity_capability_init },
    { .command = "mount",           .binary = MOUNT_BINARY,  .capinit = singularity_capability_init_default },
    { .command = "help",            .binary = MOUNT_BINARY,  .capinit = singularity_capability_init_default },
    { .command = "apps",            .binary = MOUNT_BINARY,  .capinit = singularity_capability_init_default },
    { .command = "inspect",         .binary = MOUNT_BINARY,  .capinit = singularity_capability_init_default },
    { .command = "check",           .binary = MOUNT_BINARY,  .capinit = singularity_capability_init_default },
    { .command = "image.import",    .binary = MOUNT_BINARY,  .capinit = singularity_capability_init_default },
    { .command = "image.export",    .binary = MOUNT_BINARY,  .capinit = singularity_capability_init_default },
    { .command = "instance.start",  .binary = START_BINARY,  .capinit = singularity_capability_init },
    { .command = NULL,              .binary = NULL,          .capinit = NULL }
};

int main(int argc, char **argv) {
    (void)argc;
    int index;
    char *command;
    char *binary;
    char *libexec_bin = joinpath(LIBEXECDIR, "/singularity/bin/");

    singularity_registry_init();
    singularity_config_init();
    singularity_suid_init();

    command = singularity_registry_get("COMMAND");
    if ( command == NULL ) {
        singularity_message(ERROR, "no command passed\n");
        ABORT(255);
    }

    for ( index = 0; cmd_wrapper[index].command != NULL; index++) {
        if ( strcmp(command, cmd_wrapper[index].command) == 0 ) {
            break;
        }
    }

    if ( cmd_wrapper[index].command == NULL ) {
        singularity_message(ERROR, "unknown command %s\n", command);
        ABORT(255);
    }

    /* if allow setuid is no or nosuid requested fallback to non suid command */
    if ( singularity_suid_allowed() == 0 ) {
        singularity_priv_init();
        singularity_priv_drop_perm();
    } else {
        singularity_priv_init();
        cmd_wrapper[index].capinit();
    }

    binary = strjoin(libexec_bin, cmd_wrapper[index].binary);
    execve(binary, argv, environ); // Flawfinder: ignore

    singularity_message(ERROR, "Failed to execute %s binary\n", cmd_wrapper[index].binary);
    ABORT(255);

    return(0);
}
