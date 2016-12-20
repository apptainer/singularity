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
#include <unistd.h>
#include <stdlib.h>

#include "util/file.h"
#include "util/util.h"
#include "lib/message.h"
#include "lib/privilege.h"


#include "./binds/binds.h"
#include "./home/home.h"
#include "./hostfs/hostfs.h"
#include "./kernelfs/kernelfs.h"
#include "./tmp/tmp.h"
#include "./dev/dev.h"
#include "./cwd/cwd.h"
#include "./userbinds/userbinds.h"
#include "./scratch/scratch.h"



int singularity_runtime_mount_check(void) {
    int retval = 0;

    singularity_message(VERBOSE, "Checking all mount components\n");
    retval += singularity_runtime_mount_hostfs_check();
    retval += singularity_runtime_mount_binds_check();
    retval += singularity_runtime_mount_kernelfs_check();
    retval += singularity_runtime_mount_dev_check();
    retval += singularity_runtime_mount_tmp_check();
    retval += singularity_runtime_mount_home_check();
    retval += singularity_runtime_mount_userbinds_check();
    retval += singularity_runtime_mount_scratch_check();
    retval += singularity_runtime_mount_cwd_check();

    return(retval);
}


int singularity_runtime_mount_prepare(void) {
    int retval = 0;

    singularity_message(VERBOSE, "Preparing all mount components\n");
    retval += singularity_runtime_mount_hostfs_prepare();
    retval += singularity_runtime_mount_binds_prepare();
    retval += singularity_runtime_mount_kernelfs_prepare();
    retval += singularity_runtime_mount_dev_prepare();
    retval += singularity_runtime_mount_tmp_prepare();
    retval += singularity_runtime_mount_home_prepare();
    retval += singularity_runtime_mount_userbinds_prepare();
    retval += singularity_runtime_mount_scratch_prepare();
    retval += singularity_runtime_mount_cwd_prepare();

    return(retval);
}


int singularity_runtime_mount_activate(void) {
    int retval = 0;

    singularity_message(VERBOSE, "Activating all mount components\n");
    retval += singularity_runtime_mount_hostfs_activate();
    retval += singularity_runtime_mount_binds_activate();
    retval += singularity_runtime_mount_kernelfs_activate();
    retval += singularity_runtime_mount_dev_activate();
    retval += singularity_runtime_mount_tmp_activate();
    retval += singularity_runtime_mount_home_activate();
    retval += singularity_runtime_mount_userbinds_activate();
    retval += singularity_runtime_mount_scratch_activate();
    retval += singularity_runtime_mount_cwd_activate();

    return(retval);
}


