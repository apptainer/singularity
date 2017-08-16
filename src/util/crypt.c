/*
 * Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
 *
 * See the COPYRIGHT.md file at the top-level directory of this distribution and at
 * https://github.com/singularityware/singularity/blob/master/COPYRIGHT.md.
 *
 * This file is part of the Singularity Linux container project. It is subject to the license
 * terms in the LICENSE.md file found in the top-level directory of this distribution and
 * at https://github.com/singularityware/singularity/blob/master/LICENSE.md. No part
 * of Singularity, including this file, may be copied, modified, propagated, or distributed
 * except according to the terms contained in the LICENSE.md file.
 */

#include <errno.h>
#include <stdio.h>
#include <string.h>

#include "lib/image/image.h"
#include "util/crypt.h"
#include "util/util.h"

#define SIGN_COMMAND "gpg --clearsign"

unsigned char *compute_hash(const unsigned char *data, size_t size, unsigned char *result) {
    return SHA384(data, size, result);
}

int sign_verifblock(char *verifstr, char *verifblock) {
    int ret;
    FILE *pfp;
    int p[2];
    int stdoutfd;

    singularity_message(DEBUG, "Generating signature for:\n%s\n", verifstr);

    stdoutfd = dup(1);
    if (stdoutfd < 0) {
        singularity_message(ERROR, "Could not duplicate stdout\n");
        ABORT(255);
    }

    ret = pipe(p);
    if(ret < 0){
        singularity_message(ERROR, "Could not create pipe\n");
        ABORT(255);
    }
    if (dup2(p[1], 1) < 0) {
        singularity_message(ERROR, "Could not duplicate stdout\n");
        ABORT(255);
    }
    close(p[1]);

    pfp = popen(SIGN_COMMAND, "w");
    if (pfp == NULL) {
        singularity_message(ERROR, "popen failed\n");
        ABORT(255);
    }
    if (fputs(verifstr, pfp) == EOF) {
        singularity_message(ERROR, "Could not write verifstr to pgp\n");
        ABORT(255);
    }
    if (pclose(pfp) < 0) {
        singularity_message(ERROR, "Could not close the pipe\n");
        ABORT(255);
    }
    if (dup2(stdoutfd, 1) < 0) {
        singularity_message(ERROR, "Could not duplicate stdout\n");
        ABORT(255);
    }
    close(stdoutfd);

    for ( ; ; ) {
        ret = read(p[0], verifblock, VERIFBLOCK_SIZE);
        if (ret == VERIFBLOCK_SIZE && read(p[0], verifblock, 1) != 0) {
            singularity_message(ERROR, "VB is too small to hold signature\n");
            ABORT(255);
        }
        if (ret < 0) {
            if (errno == EINTR) {
                continue;
            } else {
                singularity_message(ERROR, "read error on pipe\n");
                ABORT(255);
            }
        }
        break;
    }
    close(p[0]);

    singularity_message(DEBUG, "VB:\n%s", verifblock);
    return 0;
}
