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
#include <sys/wait.h>
#include <errno.h>
#include <string.h>


#include "config.h"
#include "config_parser.h"
#include "message.h"
#include "util.h"
#include "privilege.h"
#include "sessiondir.h"
#include "singularity.h"
#include "file.h"

#ifndef SYSCONFDIR
#define SYSCONFDIR "/etc"
#endif

int main(int argc, char **argv) {
    char *sessiondir;
    char *image = getenv("SINGULARITY_IMAGE");

    if ( image == NULL ) {
        message(ERROR, "SINGULARITY_IMAGE not defined!\n");
        ABORT(1);
    }

    config_open(joinpath(SYSCONFDIR, "/singularity/singularity.conf"));

    if ( is_suid("/proc/self/exe") == 0 ) {
        if ( is_owner(joinpath(SYSCONFDIR, "/singularity/singularity.conf"), 0 ) < 0 ) {
            message(ERROR, "Running in privileged mode, root must own the Singularity configuration file\n");
            ABORT(255);
        }

        config_rewind();
        if ( config_get_key_bool("allow setuid", 1) == 0 ) {
            message(ERROR, "Setuid mode was used, but this has been disabled by the sysadmin.\n");
            ABORT(255);
        }

        if ( getenv("SINGULARITY_NOSUID") != NULL ) {
            message(ERROR, "Requested NOSUID mode, but running as SUID.. Aborting.\n");
            ABORT(1);
        }
    } else if ( ( getenv("SINGULARITY_SUID") == NULL ) && getenv("SINGULARITY_NOSUID") == NULL ) {
        config_rewind();

        if ( config_get_key_bool("allow setuid", 1) == 1 ) {
            message(VERBOSE, "Setuid mode is allowed by the sysadmin, re-exec'ing\n");

            char sexec_path[] = LIBEXECDIR "/singularity/sexec-suid";
            setenv("SINGULARITY_SUID", "1", 1);

            execv(sexec_path, argv);
            message(ERROR, "Failed to execute sexec binary (%s): %s\n", sexec_path, strerror(errno));
            ABORT(255);
        }
    }

    priv_init();
    singularity_action_init();
    singularity_rootfs_init(image, "/var/singularity/mnt");

    sessiondir = singularity_sessiondir(image);

    message(VERBOSE, "Using sessiondir: %s\n", sessiondir);

    singularity_ns_unshare();

    singularity_rootfs_mount();

    singularity_file_create();

    singularity_mount_binds();

    singularity_mount_home();

    singularity_file_bind();

    singularity_mount_kernelfs();

    singularity_rootfs_chroot();

    singularity_action_do(argc, argv);

    return(0);

}
