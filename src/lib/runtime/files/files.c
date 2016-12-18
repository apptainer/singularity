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
#include "./passwd/passwd.h"
#include "./group/group.h"
#include "./resolvconf/resolvconf.h"


int singularity_runtime_files_check(void) {
    retval = 0;

    singularity_message(VERBOSE, "Checking all file components\n");
    retval += singularity_runtime_files_passwd_check();
    retval += singularity_runtime_files_group_check();
    retval += singularity_runtime_files_resolvconf_check();

    return(retval);
}


int singularity_runtime_files_prepare(void) {
    retval = 0;

    singularity_message(VERBOSE, "Preparing all file components\n");
    retval += singularity_runtime_files_passwd_prepare();
    retval += singularity_runtime_files_group_prepare();
    retval += singularity_runtime_files_resolvconf_prepare();

    return(retval);
}


int singularity_runtime_files_activate(void) {
    retval = 0;

    singularity_message(VERBOSE, "Activating all file components\n");
    retval += singularity_runtime_files_passwd_activate();
    retval += singularity_runtime_files_group_activate();
    retval += singularity_runtime_files_resolvconf_activate();

    return(retval);
}

