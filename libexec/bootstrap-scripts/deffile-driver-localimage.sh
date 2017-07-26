#!/bin/bash
# 
# Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
#
# This software is licensed under a 3-clause BSD license.  Please
# consult LICENSE file distributed with the sources of this project regarding
# your rights to use or distribute this software. 
#
# 

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

if [ -z "${SINGULARITY_ROOTFS:-}" ]; then
    message ERROR "Singularity root file system not defined\n"
    exit 1
fi

if [ -z "${FROM:-}" ]; then
    message ERROR "Required Definition tag 'From:' not defined.\n"
    exit 1
fi

if [ ! -f "${FROM:-}" ]; then
    message ERROR "${FROM} does not exist\n"
    exit 1
fi


########## BEGIN BOOTSTRAP SCRIPT ##########

umask 0002

message 1 "Exporting contents of ${FROM} to ${SINGULARITY_IMAGE}\n"

cmd="${SINGULARITY_libexecdir}/singularity/bin/export ${FROM} | (cd ${SINGULARITY_ROOTFS} && tar xBf -)"
if ! eval $cmd; then
    message ERROR "Failed to export contents of ${FROM} to ${SINGULARITY_ROOTFS}\n"
    ABORT 255
fi
