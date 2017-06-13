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
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <errno.h>
#include <string.h>
#include <fcntl.h>

#include "config.h"
#include "util/file.h"
#include "util/util.h"
#include "util/registry.h"
#include "lib/image/image.h"
#include "lib/runtime/runtime.h"
#include "util/privilege.h"


void daemon_registry_init(void) {
    daemon_registry_path(singularity_priv_getuid());
    
    if( is_file(singularity_registry_get("DAEMON_FILE")) ) {
        singularity_registry_set("DAEMON", "1");
    }
}

void daemon_registry_path(char *host_uid) {
    char *image_devino, *daemon_path;
    
    /* Build string with daemon file location */
    image_name = singularity_registry_get("IMAGE");
    image_devino = file_devino(image_name);
    
    daemon_path_len = strlength("/tmp/.singularity-daemon-") + strlength(host_uid) +
        strlength(image_devino) + strlength(image_name) + 3; //+3 for "/", "-", "\0"
    
    daemon_path = (char *)malloc((daemon_path_len) * sizeof(char)); 
    snprintf(daemon_path, daemon_path_len, "/tmp/.singularity-daemon-%s/%s-%s",
             host_uid, image_devino, image_name);

    /* Store daemon_file string in registry as DAEMON_FILE */
    singularity_registry_set("DAEMON_FILE", daemon_path);
    
    free(image_name);
    free(dev_ino);
    free(daemon_path);
}
