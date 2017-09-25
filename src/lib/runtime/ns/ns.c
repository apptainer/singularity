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

#include "./ipc/ipc.h"
#include "./mnt/mnt.h"
#include "./pid/pid.h"
#include "./net/net.h"
#include "../runtime.h"


int _singularity_runtime_ns(unsigned int flags) {
    int retval = 0;

    if ( flags & SR_NS_IPC ) {
        singularity_message(DEBUG, "Calling: _singularity_runtime_ns_ipc()\n");
        retval += _singularity_runtime_ns_ipc();
    }
    if ( flags & SR_NS_PID ) {
        singularity_message(DEBUG, "Calling: _singularity_runtime_ns_pid()\n");
        retval += _singularity_runtime_ns_pid();
    }
    if ( flags & SR_NS_NET ) {
        singularity_message(DEBUG, "Calling: _singularity_runtime_ns_net()\n");
        retval += _singularity_runtime_ns_net();
    }
    if ( flags & SR_NS_MNT ) {
        singularity_message(DEBUG, "Calling: _singularity_runtime_ns_mnt()\n");
        retval += _singularity_runtime_ns_mnt();
    }


    return(retval);
}

int _singularity_runtime_ns_join(unsigned int flags) {
    int retval = 0;

    if ( flags & SR_NS_IPC ) {
        singularity_message(DEBUG, "Calling: _singularity_runtime_ns_ipc_join()\n");
        retval += _singularity_runtime_ns_ipc_join();
    }
    if ( flags & SR_NS_PID ) {
        singularity_message(DEBUG, "Calling: _singularity_runtime_ns_pid_join()\n");
        retval += _singularity_runtime_ns_pid_join();
    }
    if ( flags & SR_NS_NET ) {
        singularity_message(DEBUG, "Calling: _singularity_runtime_ns_net_join()\n");
        retval += _singularity_runtime_ns_net_join();
    }
    if ( flags & SR_NS_MNT ) {
        singularity_message(DEBUG, "Calling: _singularity_runtime_ns_mnt_join()\n");
        retval += _singularity_runtime_ns_mnt_join();
    }

    return(retval);
}
