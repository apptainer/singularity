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

#define _GNU_SOURCE
#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <string.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/mount.h>
#include <sys/wait.h>
#include <unistd.h>
#include <stdlib.h>
#include <sched.h>

#include "util/file.h"
#include "util/util.h"
#include "util/registry.h"
#include "util/message.h"
#include "util/privilege.h"
#include "util/config_parser.h"

#include "./ns/ns.h"
#include "./mounts/mounts.h"
#include "./files/files.h"
#include "./enter/enter.h"
#include "./overlayfs/overlayfs.h"
#include "./environment/environment.h"
#include "./autofs/autofs.h"

#ifndef LOCALSTATEDIR
#error LOCALSTATEDIR not defined
#endif

int singularity_runtime_ns(unsigned int flags) {
    /* If a daemon already exists, join existing namespaces instead of creating */
    if ( singularity_registry_get("DAEMON_JOIN") ) {
        return(_singularity_runtime_ns_join(flags));
    }
    
    return(_singularity_runtime_ns(flags));
}

int singularity_runtime_overlayfs(void) {
    if ( singularity_registry_get("DAEMON_JOIN") ) {
        singularity_message(ERROR, "Internal Error - This function should not be called when joining an instance\n");
    }

    return(_singularity_runtime_overlayfs());
}

int singularity_runtime_environment(void) {
    return(_singularity_runtime_environment());
}

int singularity_runtime_mounts(void) {
    if ( singularity_registry_get("DAEMON_JOIN") ) {
        singularity_message(ERROR, "Internal Error - This function should not be called when joining an instance\n");
    }

    return(_singularity_runtime_mounts());
}

int singularity_runtime_files(void) {
    if ( singularity_registry_get("DAEMON_JOIN") ) {
        singularity_message(ERROR, "Internal Error - This function should not be called when joining an instance\n");
    }

    return(_singularity_runtime_files());
}

int singularity_runtime_enter(void) {
    return(_singularity_runtime_enter());
}

int singularity_runtime_autofs(void) {
    return(_singularity_runtime_autofs());
}
