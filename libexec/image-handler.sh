#!/bin/bash
# 
# Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
#
# Copyright (c) 2015-2017, Gregory M. Kurtzer. All rights reserved.
# 
# Copyright (c) 2016-2017, The Regents of the University of California,
# through Lawrence Berkeley National Laboratory (subject to receipt of any
# required approvals from the U.S. Dept. of Energy).  All rights reserved.
# 
# This software is licensed under a customized 3-clause BSD license.  Please
# consult LICENSE file distributed with the sources of this project regarding
# your rights to use or distribute this software.
# 
# NOTICE.  This Software was developed under funding from the U.S. Department of
# Energy and the U.S. Government consequently retains certain rights. As such,
# the U.S. Government has been granted for itself and others acting on its
# behalf a paid-up, nonexclusive, irrevocable, worldwide license in the Software
# to reproduce, distribute copies to the public, prepare derivative works, and
# perform publicly and display publicly, and to permit other to do so. 
# 
# 


# This script is designed to be sourced rather then executed, as a result we do
# not load functions or basic sanity.


if [ -z "${SINGULARITY_IMAGE:-}" ]; then
    message ERROR "SINGULARITY_IMAGE is undefined...\n"
    ABORT 255
fi

if [ -z "${SINGULARITY_COMMAND:-}" ]; then
    message ERROR "SINGULARITY_COMMAND is undefined...\n"
    ABORT 255
fi

case "$SINGULARITY_IMAGE" in

    instance://*)

        . "$SINGULARITY_libexecdir/singularity/handlers/image-instance.sh"

    ;;

    docker://*)
        
        . "$SINGULARITY_libexecdir/singularity/handlers/image-docker.sh"

    ;;

    http://*|https://*)

        . "$SINGULARITY_libexecdir/singularity/handlers/image-http.sh"

    ;;

    shub://*)

        . "$SINGULARITY_libexecdir/singularity/handlers/image-shub.sh"
    ;;

    *.cpioz|*.vnfs|*.cpio)

        . "$SINGULARITY_libexecdir/singularity/handlers/archive-cpio.sh"

    ;;

    *.tar|*.tgz|*.tar.gz|*.tbz|*.tar.bz)

        . "$SINGULARITY_libexecdir/singularity/handlers/archive-tar.sh"

    ;;
esac

