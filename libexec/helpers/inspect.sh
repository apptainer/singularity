#!/bin/bash
#
# Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
# Copyright (c) 2017, Vanessa Sochat. All rights reserved.
#
# See the COPYRIGHT.md file at the top-level directory of this distribution and at
# https://github.com/singularityware/singularity/blob/master/COPYRIGHT.md.
#
# This file is part of the Singularity Linux container project. It is subject to the license
# terms in the LICENSE.md file found in the top-level directory of this distribution and
# at https://github.com/singularityware/singularity/blob/master/LICENSE.md. No part
# of Singularity, including this file, may be copied, modified, propagated, or distributed
# except according to the terms contained in the LICENSE.md file.


## Basic sanity
if [ -z "$SINGULARITY_libexecdir" ]; then
    echo "Could not identify the Singularity libexecdir."
    exit 1
fi

## Load functions
if [ -f "$SINGULARITY_libexecdir/singularity/functions" ]; then
    . "$SINGULARITY_libexecdir/singularity/functions"
else
    echo "Error loading functions: $SINGULARITY_libexecdir/singularity/functions"
    exit 1
fi

if [ ! -d "${SINGULARITY_MOUNTPOINT}" ]; then
    message ERROR "The mount point does not exist: ${SINGULARITY_MOUNTPOINT}\n"
    ABORT 255
fi

if [ ! -d "${SINGULARITY_MOUNTPOINT}/.singularity.d" ]; then
    message ERROR "The Singularity metadata directory does not exist in image\n"
    ABORT 255
fi

SINGULARITY_ROOTFS=${SINGULARITY_MOUNTPOINT}
export SINGULARITY_MOUNTPOINT SINGULARITY_INSPECT_LABELS SINGULARITY_INSPECT_DEFFILE SINGULARITY_INSPECT_RUNSCRIPT SINGULARITY_INSPECT_TEST SINGULARITY_INSPECT_ENVIRONMENT SINGULARITY_ROOTFS SINGULARITY_PRINT_STRUCTURED SINGULARITY_INSPECT_HELP

if [ ! -z ${SINGULARITY_APPNAME+x} ]; then
    export SINGULARITY_APPNAME
fi

eval_abort "$SINGULARITY_libexecdir/singularity/python/helpers/json/inspect.py"
