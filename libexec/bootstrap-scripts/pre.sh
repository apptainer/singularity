#!/bin/bash
#
# Copyright (c) 2017-2018, SyLabs, Inc. All rights reserved.
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
if [ -f "$SINGULARITY_libexecdir/singularity/bootstrap-scripts/functions" ]; then
    . "$SINGULARITY_libexecdir/singularity/bootstrap-scripts/functions"
else
    echo "Error loading functions: $SINGULARITY_libexecdir/singularity/bootstrap-scripts/functions"
    exit 1
fi


if [ -z "${SINGULARITY_ROOTFS:-}" ]; then
    message ERROR "Singularity root file system not defined\n"
    exit 1
fi

umask 0002

install -d -m 0755 "$SINGULARITY_ROOTFS"
install -d -m 0755 "$SINGULARITY_ROOTFS/.singularity.d"
install -d -m 0755 "$SINGULARITY_ROOTFS/.singularity.d/env"

exit 0
