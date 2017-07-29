#!/bin/bash
# 
# Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
# Copyright (c) 2015-2017, Gregory M. Kurtzer. All rights reserved.
# Copyright (c) 2017, Vanessa Sochat. All rights reserved.


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
export SINGULARITY_MOUNTPOINT SINGULARITY_ROOTFS

eval_abort "$SINGULARITY_libexecdir/singularity/python/helpers/printer/help.py" --file "${SINGULARITY_MOUNTPOINT}/.singularity.d/runscript.help"
