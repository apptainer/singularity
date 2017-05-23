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
    message ERROR "Singularity root file system not defined\n"
    exit 1
fi

message 1 "Bootstrap initialization\n"

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
    message 1 "Checking bootstrap definition\n";
    if [ -f "$SINGULARITY_BUILDDEF" ]; then

        ### Obtain the BootStrap build type from the SPEC
        if SINGULARITY_OSBUILD=`singularity_key_get "BootStrap" "$SINGULARITY_BUILDDEF"`; then
            true
        elif SINGULARITY_OSBUILD=`singularity_key_get "OSBuild" "$SINGULARITY_BUILDDEF"`; then
            message WARNING "the key 'OSBuild' has been superseded by 'BootStrap'\n"
        elif SINGULARITY_OSBUILD=`singularity_key_get "OSType" "$SINGULARITY_BUILDDEF"`; then
            message WARNING "the key 'OSType' has been superseded by 'BootStrap'\n"
        else
            message 1 "No 'BootStrap' build module given, assuming overlay\n"
        fi

        export SINGULARITY_BUILDDEF SINGULARITY_OSBUILD
    else
        message ERROR "Build Definition file not found: $SINGULARITY_BUILDDEF\n"
        exit 1
    fi
else
    message 1 "No bootstrap definition passed, updating container\n"
fi


if [ -f "$SINGULARITY_libexecdir/singularity/bootstrap/modules-v2/prebootstrap.sh" ]; then
    message 1 "Executing Prebootstrap module\n"
    if ! eval "$SINGULARITY_libexecdir/singularity/bootstrap/modules-v2/prebootstrap.sh" "$@"; then
        exit 255
    fi
else
    message ERROR "Could not locate Prebootstrap module\n"
    exit 255
fi


if [ -n "${SINGULARITY_OSBUILD:-}" ]; then
    if [ -f "$SINGULARITY_libexecdir/singularity/bootstrap/modules-v2/build-$SINGULARITY_OSBUILD.sh" ]; then
        if [ -x "$SINGULARITY_ROOTFS/bin/sh" -a -z "${SINGULARITY_REBOOTSTRAP:-}" ]; then
            message 1 "Not bootstrapping core container\n"
        else
            message 1 "Executing Bootstrap '$SINGULARITY_OSBUILD' module\n"
            if ! eval "$SINGULARITY_libexecdir/singularity/bootstrap/modules-v2/build-$SINGULARITY_OSBUILD.sh" "$@"; then
                exit 255
            fi
        fi
    else
        message ERROR "Unrecognized OSBuild type: $SINGULARITY_OSBUILD\n"
        exit 255
    fi
fi


if [ -f "$SINGULARITY_libexecdir/singularity/bootstrap/modules-v2/postbootstrap.sh" ]; then
    message 1 "Executing Postbootstrap module\n"
    if ! eval "$SINGULARITY_libexecdir/singularity/bootstrap/modules-v2/postbootstrap.sh" "$@"; then
        exit 255
    fi
else
    message ERROR "Could not locate Postbootstrap module"
    exit 255
fi

message 1 "Done.\n"
exit 0
