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

#define _GNU_SOURCE
#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <string.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/mount.h>
#include <unistd.h>
#include <stdlib.h>
#include <sched.h>

#include "file.h"
#include "util.h"
#include "message.h"
#include "config_parser.h"
#include "privilege.h"

static int userns_enabled = 0;


int singularity_ns_user_unshare(void) {
    config_rewind();

#ifdef NS_CLONE_NEWUSER
    message(DEBUG, "Attempting to virtualize the USER namespace\n");
    if ( unshare(CLONE_NEWUSER) == 0 ) {
        message(DEBUG, "Enabling user namespaces\n");
        userns_enabled = 1;
    } else {
        message(VERBOSE3, "User namespaces not supported\n");
    }
#else
    message(VERBOSE3, "Not virtualizing USER namespace (no host support)\n");
#endif

    return(0);
}


int singularity_ns_user_enabled(void) {
    return(userns_enabled);
}
