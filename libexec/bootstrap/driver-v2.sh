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
    messge ERROR "Singularity root file system not defined\n"
    exit 1
fi


if [ -z "${LC_ALL:-}" ]; then
    LC_ALL=C
fi
if [ -z "${LANG:-}" ]; then
    LANG=C
fi
if [ -z "${TERM:-}" ]; then
    TERM=xterm
fi
DEBIAN_FRONTEND=noninteractive
export LC_ALL LANG TERM DEBIAN_FRONTEND


if [ -n "${SINGULARITY_BUILDDEF:-}" ]; then
    if [ -f "$SINGULARITY_BUILDDEF" ]; then
        ### Obtain the DistType from the SPEC
        SINGULARITY_DISTTYPE=`singularity_key_get "DistType" "$SINGULARITY_BUILDDEF"`
        if [ -z "${SINGULARITY_DISTTYPE:-}" ]; then
            echo "DistType: Requires an argument!" 2>&2
            exit 1
        fi
        export SINGULARITY_BUILDDEF SINGULARITY_DISTTYPE
    else
        message ERROR "Build Definition file not found: $SINGULARITY_BUILDDEF\n"
        exit 1
    fi
fi


if [ -f "$SINGULARITY_libexecdir/singularity/bootstrap/modules-v2/prebootstrap.sh" ]; then
    if ! eval "$SINGULARITY_libexecdir/singularity/bootstrap/modules-v2/prebootstrap.sh" "$@"; then
        exit 255
    fi
else
    message ERROR "Could not locate pre Bootstrap module"
    exit 255
fi


if [ -n "${SINGULARITY_DISTTYPE:-}" ]; then
    if [ -f "$SINGULARITY_libexecdir/singularity/bootstrap/modules-v2/dist-$SINGULARITY_DISTTYPE.sh" ]; then
        if ! eval "$SINGULARITY_libexecdir/singularity/bootstrap/modules-v2/dist-$SINGULARITY_DISTTYPE.sh" "$@"; then
            exit 255
        fi
    else
        message ERROR "Unrecognized Distribution type: $SINGULARITY_DISTTYPE\n"
        exit 255
    fi
fi


if [ -f "$SINGULARITY_libexecdir/singularity/bootstrap/modules-v2/setup.sh" ]; then
    if ! eval "$SINGULARITY_libexecdir/singularity/bootstrap/modules-v2/setup.sh" "$@"; then
        exit 255
    fi
else
    message ERROR "Could not locate setup Bootstrap module"
    exit 255
fi



if [ -f "$SINGULARITY_libexecdir/singularity/bootstrap/modules-v2/postbootstrap.sh" ]; then
    if ! eval "$SINGULARITY_libexecdir/singularity/bootstrap/modules-v2/postbootstrap.sh" "$@"; then
        exit 255
    fi
else
    message ERROR "Could not locate post Bootstrap module"
    exit 255
fi

exit 0
