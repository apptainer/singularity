#!/bin/bash
# 
# Copyright (c) 2015-2016, Gregory M. Kurtzer. All rights reserved.
# 
# “Singularity” Copyright (c) 2016, The Regents of the University of California,
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
    message ERROR "SINGULARITY_ROOTFS is undefined!\n"
    exit 255
fi

if [ ! -d  "$SINGULARITY_ROOTFS" ]; then
    message ERROR "SINGULARITY_ROOTFS is not a valid directory!\n"
    exit 255
fi


if [ -n "${SINGULARITY_EXPORT_COMMAND:-}" ]; then
    eval "(cd $SINGULARITY_ROOTFS; $SINGULARITY_EXPORT_COMMAND)"
else
    if [ -n "${SINGULARITY_EXPORT_FILE:-}" ]; then
        eval "(cd $SINGULARITY_ROOTFS; tar -c .) > $SINGULARITY_EXPORT_FILE"
    else
        eval "(cd $SINGULARITY_ROOTFS; tar -c .)"
    fi
fi
