#!/bin/bash
# 
# Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
# 
# See the COPYRIGHT.md file at the top-level directory of this distribution and at
# https://github.com/singularityware/singularity/blob/master/COPYRIGHT.md.
# 
# This file is part of the Singularity Linux container project. It is subject to the license
# terms in the LICENSE.md file found in the top-level directory of this distribution and
# at https://github.com/singularityware/singularity/blob/master/LICENSE.md. No part
# of Singularity, including this file, may be copied, modified, propagated, or distributed
# except according to the terms contained in the LICENSE.md file.
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
elif [ ! -e "${FROM:-}" ]; then
    message ERROR "${FROM} does not exist\n"
    exit 1
fi


########## BEGIN BOOTSTRAP SCRIPT ##########

umask 0002

message 1 "Exporting contents of ${FROM} to ${SINGULARITY_IMAGE}\n"
if ! eval "${SINGULARITY_bindir}"/singularity image.export "${FROM}" | tar xBf - -C "${SINGULARITY_ROOTFS}"; then
    message ERROR "Failed to export contents of ${FROM} to ${SINGULARITY_ROOTFS}\n"
    ABORT 255
fi
