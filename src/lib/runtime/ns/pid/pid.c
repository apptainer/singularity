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
#include "util/message.h"
#include "util/config_parser.h"
#include "util/privilege.h"
#include "util/fork.h"
#include "util/registry.h"


int _singularity_runtime_ns_pid(void) {
    if ( singularity_config_get_bool(ALLOW_PID_NS) <= 0 ) {
        singularity_message(VERBOSE2, "Not virtualizing PID namespace by configuration\n");
        return(0);
    }

    if ( singularity_registry_get("UNSHARE_PID") == NULL ) {
        singularity_message(VERBOSE2, "Not virtualizing PID namespace on user request\n");
        return(0);
    }

#ifdef NS_CLONE_NEWPID
    singularity_message(DEBUG, "Using PID namespace: CLONE_NEWPID\n");
    
#else
#ifdef NS_CLONE_PID
    singularity_message(DEBUG, "Using PID namespace: CLONE_PID\n");

#else
    singularity_message(WARNING, "Skipping PID namespace creation, support not available on host\n");
    return(0);

#endif
#endif

    singularity_message(DEBUG, "Virtualizing PID namespace\n");
    singularity_fork_run(CLONE_NEWPID);

    singularity_registry_set("PIDNS_ENABLED", "1");

    return(0);
}
