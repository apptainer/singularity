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

#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <errno.h> 
#include <string.h>
#include <fcntl.h>  
#include <linux/limits.h>

#include "util/message.h"
#include "util/util.h"
#include "util/file.h"
#include "util/registry.h"

#include "../image.h"

#define BUFFER_SIZE     (1024*1024)
#define MAX_LINE_LEN    2048

#define EXTFS_MAGIC "\123\357"

#define COMPAT_HASJOURNAL 0x4

#define INCOMPAT_FILETYPE 0x2
#define INCOMPAT_RECOVER 0x4
#define INCOMPAT_METABG 0x10

#define ROCOMPAT_SPARSESUPER 0x1
#define ROCOMPAT_LARGEFILE 0x2
#define ROCOMPAT_BTREEDIR 0x4

struct extfs_info {
	unsigned char	magic[2];
	uint16_t	state;
	uint32_t	dummy[8];
	uint32_t	feat_compat;
	uint32_t	feat_incompat;
	uint32_t	feat_rocompat;
};


int _singularity_image_ext3_init(struct image_object *image, int open_flags) {
    int image_fd;
    int ret;
    int magicoff = 1080;
    FILE *image_fp;
    static char buf[2048];
    struct extfs_info *einfo;

    singularity_message(DEBUG, "Opening file descriptor to image: %s\n", image->path);
    if ( ( image_fd = open(image->path, open_flags, 0755) ) < 0 ) {
        singularity_message(ERROR, "Could not open image %s: %s\n", image->path, strerror(errno));
        ABORT(255);
    }

    if ( ( image_fp = fdopen(dup(image_fd), "r") ) == NULL ) {
        singularity_message(ERROR, "Could not associate file pointer from file descriptor on image %s: %s\n", image->path, strerror(errno));
        ABORT(255);
    }


    singularity_message(VERBOSE3, "Checking that file pointer is a Singularity image\n");
    rewind(image_fp);

    // Get the first line from the config
    ret = fread(buf, 1, sizeof(buf), image_fp);
    fclose(image_fp);
    if ( ret != sizeof(buf) ) {
        singularity_message(DEBUG, "Could not read the top of the image\n");
        return(-1);
    }

    /* if LAUNCH_STRING is present, figure out EXTFS magic offset */
    if ( strstr(buf, "singularity") != NULL ) {
        magicoff += strlen(buf);
        image->offset = strlen(buf);
    }

    einfo = (struct extfs_info *)&buf[magicoff];
    if ( memcmp(einfo->magic, EXTFS_MAGIC, 2 ) != 0 ) {
        close(image_fd);
        singularity_message(VERBOSE, "File is not a valid EXT3 image\n");
        return(-1);
    }
    /* Check for features supported by EXT3 */
    if ( !(einfo->feat_compat & COMPAT_HASJOURNAL) ) {
        close(image_fd);
        singularity_message(VERBOSE, "File is not a valid EXT3 image\n");
        return(-1);
    }
    /* check for unsupported incompat ext3 features */
    if ( einfo->feat_incompat & ~(INCOMPAT_FILETYPE|INCOMPAT_RECOVER|INCOMPAT_METABG) ) {
        close(image_fd);
        singularity_message(VERBOSE, "File is not a valid EXT3 image\n");
        return(-1);
    }
    /* check for unsupported rocompat ext3 features */
    if ( einfo->feat_rocompat & ~(ROCOMPAT_SPARSESUPER|ROCOMPAT_LARGEFILE|ROCOMPAT_BTREEDIR) ) {
        close(image_fd);
        singularity_message(VERBOSE, "File is not a valid EXT3 image\n");
        return(-1);
    }

    image->fd = image_fd;

    return(0);
}
