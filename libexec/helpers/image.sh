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

if ! which tar >/dev/null; then
    message ERROR "Could not find the program: tar\n"
    ABORT 255
fi


case "$SINGULARITY_COMMAND" in
    image.import) 
        if [ -n "${SINGULARITY_IMPORT_FILE:-}" ]; then
            exec zcat "${SINGULARITY_IMPORT_FILE}" | tar --ignore-failed-read -xf - -C "$SINGULARITY_MOUNTPOINT"
        else
            exec tar --ignore-failed-read -xf - -C "$SINGULARITY_MOUNTPOINT"
        fi
    ;;
    image.export)
        if [ -n "${SINGULARITY_EXPORT_FILE:-}" ]; then
            exec tar --ignore-failed-read -cf - -C "$SINGULARITY_MOUNTPOINT" . > "${SINGULARITY_EXPORT_FILE}"
        else
            exec tar --ignore-failed-read -cf - -C "$SINGULARITY_MOUNTPOINT" .
        fi
    ;;
    *)
        message ERROR "Unknown image class command\n"
        ABORT 255
    ;;
esac
