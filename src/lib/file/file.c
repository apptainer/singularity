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
#include "passwd/passwd.h"
#include "group/group.h"
#include "resolvconf/resolvconf.h"
#include "entrypoint/entrypoint.h"



int singularity_file(void) {
    int retval = 0;

    retval += singularity_file_passwd();
    retval += singularity_file_group();
    retval += singularity_file_resolvconf();

    return(retval);
}

int singularity_file_bootstrap(void) {
    int retval = 0;

    retval += singularity_file_passwd();
    retval += singularity_file_group();
    retval += singularity_file_resolvconf();
    retval += singularity_file_entrypoint("run");
    retval += singularity_file_entrypoint("exec");
    retval += singularity_file_entrypoint("shell");

    return(retval);
}
