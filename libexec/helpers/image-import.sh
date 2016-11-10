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

IMPORT_URI="${1:-}"

if [ -z "$IMPORT_URI" ]; then
    message 1 "Assuming import from incoming pipe\n"
    IMPORT_URI="-"
fi

case "$IMPORT_URI" in
    docker://*)
        CONTAINER_NAME=`echo "$IMPORT_URI" | sed -e 's@^docker://@@'`
        SINGULARITY_IMPORT_GET="$SINGULARITY_libexecdir/singularity/python/cli.py --rootfs '$SINGULARITY_ROOTFS' --docker '$CONTAINER_NAME' --cmd"
    ;;
    http://*|https://*)
        SINGULARITY_IMPORT_GET="curl -L -k '$IMPORT_URI'"
    ;;
    file://*)
        LOCAL_FILE=`echo "$IMPORT_URI" | sed -e 's@^file://@@'`
        if [ ! -f "$LOCAL_FILE" ]; then
            message ERROR "URI file not found: $LOCAL_FILE\n"
            ABORT 1
        fi
        SINGULARITY_IMPORT_GET="cat '$LOCAL_FILE'"
    ;;
    *://*)
        message ERROR "Unsupported URI: $IMPORT_URI\n"
        ABORT 1
    ;;
    -)
        SINGULARITY_IMPORT_GET="cat"
    ;;
    *)
        if [ ! -f "$IMPORT_URI" ]; then
            message ERROR "File not found: $IMPORT_URI\n"
            ABORT 1
        fi
        SINGULARITY_IMPORT_GET="cat $IMPORT_URI"
    ;;
esac

case "$IMPORT_URI" in
    *.tar|-)
        SINGULARITY_IMPORT_SPLAT="| ( cd '$SINGULARITY_ROOTFS' ; tar -xf - )"
    ;;
    *.tar.gz|*.tgz)
        SINGULARITY_IMPORT_SPLAT="| ( cd '$SINGULARITY_ROOTFS' ; tar -xzf - )"
    ;;
    *.tar.bz2)
        SINGULARITY_IMPORT_SPLAT="| ( cd '$SINGULARITY_ROOTFS' ; tar -xjf - )"
    ;;
esac

eval "$SINGULARITY_IMPORT_GET ${SINGULARITY_IMPORT_SPLAT:-}"

ret_val=$?
if [ $ret_val != 0 ];then
    ABORT $ret_val
fi

eval "$SINGULARITY_libexecdir/singularity/bootstrap/main.sh"

exit $?
