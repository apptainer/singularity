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
    char *image;

    // Before we do anything, check privileges and drop permission
    priv_init();
    priv_drop();

#ifdef SINGULARITY_SUID
    message(VERBOSE2, "Running SUID program workflow\n");

    message(VERBOSE2, "Checking program has appropriate permissions\n");
    if ( ( is_owner("/proc/self/exe", 0 ) < 0 ) || ( is_suid("/proc/self/exe") < 0 ) ) {
        singularity_abort(255, "This program must be SUID root\n");
    }

    message(VERBOSE2, "Checking configuration file is properly owned by root\n");
    if ( is_owner(joinpath(SYSCONFDIR, "/singularity/singularity.conf"), 0 ) < 0 ) {
        singularity_abort(255, "Running in privileged mode, root must own the Singularity configuration file\n");
    }

    config_open(joinpath(SYSCONFDIR, "/singularity/singularity.conf"));

    config_rewind();
    
    message(VERBOSE2, "Checking that we are allowed to run as SUID\n");
    if ( config_get_key_bool("allow setuid", 0) == 0 ) {
        singularity_abort(255, "SUID mode has been disabled by the sysadmin... Aborting\n");
    }

    message(VERBOSE2, "Checking if we were requested to run as NOSUID by user\n");
    if ( getenv("SINGULARITY_NOSUID") != NULL ) {
        singularity_abort(1, "NOSUID mode has been requested... Aborting\n");
    }

#else
    message(VERBOSE, "Running NON-SUID program workflow\n");

    message(DEBUG, "Checking program has appropriate permissions\n");
    if ( is_suid("/proc/self/exe") >= 0 ) {
        singularity_abort(255, "This program must **NOT** be SUID\n");
    }

    config_open(joinpath(SYSCONFDIR, "/singularity/singularity.conf"));

    config_rewind();

    if ( priv_getuid() != 0 ) {
        message(VERBOSE2, "Checking that we are allowed to run as SUID\n");
        if ( config_get_key_bool("allow setuid", 0) == 1 ) {
            message(VERBOSE2, "Checking if we were requested to run as NOSUID by user\n");
            if ( getenv("SINGULARITY_NOSUID") == NULL ) {
                char sexec_suid_path[] = LIBEXECDIR "/singularity/sexec-suid";

                if ( ( is_owner(sexec_suid_path, 0 ) == 0 ) && ( is_suid(sexec_suid_path) == 0 ) ) {
                    message(VERBOSE, "Invoking SUID sexec: %s\n", sexec_suid_path);

                    execv(sexec_suid_path, argv);
                    singularity_abort(255, "Failed to execute sexec binary (%s): %s\n", sexec_suid_path, strerror(errno));
                } else {
                    message(VERBOSE, "Not invoking SUID mode: SUID sexec not installed\n");
                }
            } else {
                message(VERBOSE, "Not invoking SUID mode: NOSUID mode requested\n");
            }
        } else {
            message(VERBOSE, "Not invoking SUID mode: disallowed by the system administrator\n");
        }
    } else {
        message(VERBOSE, "Not invoking SUID mode: running as root\n");
    }

#endif /* SINGULARITY_SUID */

    if ( ( image = getenv("SINGULARITY_IMAGE") ) == NULL ) {
        singularity_abort(255, "SINGULARITY_IMAGE not defined!\n");
    }

    singularity_action_init();
    singularity_rootfs_init(image, "/var/singularity/mnt");

    sessiondir = singularity_sessiondir(image);

    message(VERBOSE, "Using sessiondir: %s\n", sessiondir);

    singularity_ns_unshare();

    singularity_rootfs_mount();

    singularity_file();

    singularity_mount();

    singularity_rootfs_chroot();

    singularity_action_do(argc, argv);

    return(0);

}
