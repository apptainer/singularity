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

#include "config.h"
#include "ns/ns.h"
#include "rootfs/rootfs.h"

int singularity_init(void) {
    int retval = 0;

//    retval += ns_init();

    return(retval);
}

/*
int singularity_ns_pid_unshare(void) {
    return(ns_pid_unshare());
}
int singularity_ns_mnt_unshare(void) {
    return(ns_mnt_unshare());
}
int singularity_ns_join(pid_t attach_pid) {
    return(ns_join(attach_pid));
}
*/
