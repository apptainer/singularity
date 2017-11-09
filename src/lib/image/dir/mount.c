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

#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <string.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/file.h>
#include <sys/mount.h>
#include <unistd.h>
#include <stdlib.h>

#include "util/file.h"
#include "util/util.h"
#include "util/message.h"
#include "util/config_parser.h"
#include "util/privilege.h"
#include "util/mount.h"

#include "../image.h"


int _singularity_image_dir_mount(struct image_object *image, char *mount_point) {

    if ( strcmp(image->path, "/") == 0 ) {
        singularity_message(ERROR, "Naughty naughty naughty...\n");
        ABORT(255);
    }

    singularity_priv_escalate();
    singularity_message(DEBUG, "Mounting container directory %s->%s\n", image->path, mount_point);
    if ( singularity_mount(image->path, mount_point, NULL, MS_BIND|MS_NOSUID|MS_REC|MS_NODEV, NULL) < 0 ) {
        singularity_message(ERROR, "Could not mount container directory %s->%s: %s\n", image->path, mount_point, strerror(errno));
        return 1;
    }
    singularity_priv_drop();

    return(0);
}

