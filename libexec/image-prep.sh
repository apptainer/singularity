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


if [ -z "${SINGULARITY_IMAGE:-}" ]; then
    message ERROR "SINGULARITY_IMAGE is undefined...\n"
    ABORT 255
fi

if [ -z "${SINGULARITY_COMMAND:-}" ]; then
    message ERROR "SINGULARITY_COMMAND is undefined...\n"
    ABORT 255
fi


# Don't trust environment
USERID=`id -u`
HOMEDIR=`getent passwd $USERID | cut -d: -f6`

case "$SINGULARITY_IMAGE" in
    http://*|https://*)
        NAME=`basename "$SINGULARITY_IMAGE"`
        if [ -f "$NAME" ]; then
            message 2 "Using cached container in current working directory: $NAME\n"
            SINGULARITY_IMAGE="$NAME"
        else
            message 1 "Caching container to current working directory: $NAME\n"
            if curl -L -k "$SINGULARITY_IMAGE" > "$NAME"; then
                SINGULARITY_IMAGE="$NAME"
            else
                ABORT 255
            fi
        fi
    ;;
esac

case "$SINGULARITY_IMAGE" in
    *.tgz|*.tar.gz)
        NAME=`basename "$SINGULARITY_IMAGE"`
        TIMESTAMP=`stat -c "%Y" "$SINGULARITY_IMAGE"`
        CONTAINER_DIR="$HOMEDIR/.singularity/cache/$NAME/$TIMESTAMP/$NAME"
        if [ ! -d "$CONTAINER_DIR" ]; then
            mkdir -p "$CONTAINER_DIR"
            tar -C "$CONTAINER_DIR" -xzf "$SINGULARITY_IMAGE" 2>/dev/null # this almost always gives permission errors
        fi
        SINGULARITY_IMAGE="$CONTAINER_DIR"
    ;;
    *.tbz|*.tar.bz)
        NAME=`basename "$SINGULARITY_IMAGE"`
        TIMESTAMP=`stat -c "%Y" "$SINGULARITY_IMAGE"`
        CONTAINER_DIR="$HOMEDIR/.singularity/cache/$NAME/$TIMESTAMP/$NAME"
        if [ ! -d "$CONTAINER_DIR" ]; then
            mkdir -p "$CONTAINER_DIR"
            tar -C "$CONTAINER_DIR" -xjf "$SINGULARITY_IMAGE" 2>/dev/null # this almost always gives permission errors
        fi
        SINGULARITY_IMAGE="$CONTAINER_DIR"
    ;;
esac

