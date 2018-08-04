/* 
 * Copyright (c) 2017-2018, SyLabs, Inc. All rights reserved.
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

#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <string.h>
#include <sys/stat.h>
#include <unistd.h>
#include <stdlib.h>

#include "util/file.h"
#include "util/util.h"
#include "util/message.h"
#include "util/mountlist.h"
#include "util/privilege.h"

#include "./binds/binds.h"
#include "./home/home.h"
#include "./hostfs/hostfs.h"
#include "./kernelfs/kernelfs.h"
#include "./tmp/tmp.h"
#include "./dev/dev.h"
#include "./cwd/cwd.h"
#include "./userbinds/userbinds.h"
#include "./scratch/scratch.h"
#include "./libs/libs.h"
#include "./domounts/domounts.h"


int _singularity_runtime_mounts(void) {
    int retval = 0;
    struct mountlist mountlist;
    memset(&mountlist, 0, sizeof(mountlist));

    singularity_message(VERBOSE, "Running all mount components\n");
    retval += _singularity_runtime_mount_dev(&mountlist);
    retval += _singularity_runtime_mount_kernelfs(&mountlist);
    retval += _singularity_runtime_mount_hostfs(&mountlist);
    retval += _singularity_runtime_mount_binds(&mountlist);
    retval += _singularity_runtime_mount_home(&mountlist);
    retval += _singularity_runtime_mount_userbinds(&mountlist);
    retval += _singularity_runtime_mount_tmp(&mountlist);
    retval += _singularity_runtime_mount_scratch(&mountlist);
    retval += _singularity_runtime_mount_cwd(&mountlist);
    retval += _singularity_runtime_mount_libs(&mountlist);

    retval += _singularity_runtime_domounts(&mountlist);

    mountlist_cleanup(&mountlist);

    return(retval);
}

