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
#include <sys/mount.h>
#include <unistd.h>
#include <stdlib.h>

#include "file.h"
#include "util.h"
#include "message.h"
#include "privilege.h"
#include "config_parser.h"
#include "ns/ns.h"


int singularity_mount_kernelfs() {

    //if ( ( singularity_ns_pid_enabled() < 0 ) && ( singularity_ns_user_enabled() < 0 ) ) {
    if ( singularity_ns_pid_enabled() < 0 ) {
        message(VERBOSE, "Not mounting kernel file systems, pid namespaces not enabled\n");
        return(0);
    }

    // Mount /proc if we are configured
    message(DEBUG, "Checking configuration file for 'mount proc'\n");
    config_rewind();
    if ( config_get_key_bool("mount proc", 1) > 0 ) {
        if ( is_dir("/proc") == 0 ) {
            priv_escalate();
            message(VERBOSE, "Mounting /proc\n");
            if ( mount("proc", "/proc", "proc", 0, NULL) < 0 ) {
                message(ERROR, "Could not mount /proc: %s\n", strerror(errno));
                ABORT(255);
            }
            priv_drop();
        } else {
            message(WARNING, "Not mounting /proc, container has no bind directory\n");
        }
    } else {
        message(VERBOSE, "Skipping /proc mount\n");
    }

    if ( singularity_ns_user_enabled() >= 0 ) {
        message(VERBOSE, "Not mounting /sys, user namespace in use\n");
        return(0);
    }

    // Mount /sys if we are configured
    message(DEBUG, "Checking configuration file for 'mount sys'\n");
    config_rewind();
    if ( config_get_key_bool("mount sys", 1) > 0 ) {
        if ( is_dir("/sys") == 0 ) {
            priv_escalate();
            message(VERBOSE, "Mounting /sys\n");
            if ( mount("sysfs", "/sys", "sysfs", 0, NULL) < 0 ) {
                message(ERROR, "Could not mount /sys: %s\n", strerror(errno));
                ABORT(255);
            }
            priv_drop();
        } else {
            message(WARNING, "Not mounting /sys, container has no bind directory\n");
        }
    } else {
        message(VERBOSE, "Skipping /sys mount\n");
    }

    return(0);
}
