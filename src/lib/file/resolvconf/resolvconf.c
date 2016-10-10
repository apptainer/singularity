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
#include <sys/types.h>
#include <limits.h>
#include <unistd.h>
#include <stdlib.h>
#include <grp.h>
#include <pwd.h>


#include "util/file.h"
#include "util/util.h"
#include "lib/config_parser.h"
#include "lib/message.h"
#include "lib/privilege.h"
#include "lib/sessiondir.h"
#include "lib/rootfs/rootfs.h"
#include "lib/file/file-bind.h"


int singularity_file_resolvconf(void) {
    char *file = "/etc/resolv.conf";

    singularity_message(DEBUG, "Checking configuration option\n");
    singularity_config_rewind();
    if ( singularity_config_get_bool("config resolv_conf", 1) <= 0 ) {
        singularity_message(VERBOSE, "Skipping bind of the host's %s\n", file);
        return(0);
    }

    container_file_bind(file, file);

    return(0);
}
