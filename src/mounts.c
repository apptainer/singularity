/* 
 * Copyright (c) 2015-2016, Gregory M. Kurtzer. All rights reserved.
 * 
 * “Singularity” Copyright (c) 2016, The Regents of the University of California,
 * through Lawrence Berkeley National Laboratory (subject to receipt of any
 * required approvals from the U.S. Dept. of Energy).  All rights reserved.
 * 
 * If you have questions about your rights to use or distribute this software,
 * please contact Berkeley Lab's Innovation & Partnerships Office at
 * IPO@lbl.gov.
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
#include <sys/mount.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <errno.h> 
#include <string.h>
#include <fcntl.h>

#include "config.h"
#include "mounts.h"
#include "util.h"
#include "loop-control.h"


int mount_image(char * image_path, char * mount_point) {
    char * loop_device;
    uid_t uid = getuid();
    gid_t gid = getgid();

    if ( s_is_file(image_path) < 0 ) {
        fprintf(stderr, "ERROR: Could not access image file: %s\n", image_path);
        return(-1);
    }

    if ( s_is_dir(mount_point) < 0 ) {
        fprintf(stderr, "ERROR: Mount point is not available: %s\n", mount_point);
        return(-1);
    }

    if ( obtain_loop_dev(&loop_device) < 0 ) {
        fprintf(stderr, "FAILED: Could not obtain loop device\n");
        return(-1);
    }

    if ( associate_loop_dev(image_path, loop_device) < 0 ) {
        fprintf(stderr, "FAILED: Could not associate loop device\n");
        return(-1);
    }

    printf("Mounting image to %s\n", mount_point);

    if ( mount(loop_device, mount_point, "ext4", MS_NOSUID, "discard") < 0 ) {
        fprintf(stderr, "ERROR: Failed to mount '%s' at '%s': %s\n", loop_device, mount_point, strerror(errno));
        return(-1);
    }

    return(0);
}


