/* 
 * Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
 * 
 * This software is licensed under a 3-clause BSD license.  Please
 * consult LICENSE file distributed with the sources of this project regarding
 * your rights to use or distribute this software.
 * 
 */


#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <errno.h>
#include <string.h>
#include <fcntl.h>
#include <libgen.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <sys/un.h>
#include <sys/wait.h>

#include "config.h"
#include "util/file.h"
#include "util/fork.h"
#include "util/util.h"
#include "util/daemon.h"
#include "util/registry.h"
#include "lib/image/image.h"
#include "lib/runtime/runtime.h"
#include "util/config_parser.h"
#include "util/fork.h"
#include "util/privilege.h"
#include "util/suid.h"
#include "util/sessiondir.h"
#include "util/cleanupd.h"

#include "./action-lib/include.h"

#ifndef SYSCONFDIR
#error SYSCONFDIR not defined
#endif


int main(int argc, char **argv) {

    if (chdir("/") < 0 ) {
        singularity_message(ERROR, "Can't change directory to /\n");
    }
    setsid();
    umask(0);

    while(1) {
        pause();
    }
    
    return(0);
}
