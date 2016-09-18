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


# This script is designed to be sourced rather then executed, as a result we do
# not load functions or basic sanity.


if [ -z "${SINGULARITY_IMAGE:-}" ]; then
    message ERROR "SINGULARITY_IMAGE is undefined...\n"
    ABORT 255
fi

if [ -z "${SINGULARITY_COMMAND:-}" ]; then
    message ERROR "SINGULARITY_COMMAND is undefined...\n"
    ABORT 255
fi

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
    docker://*)
        echo
        echo "A hint of things to come..."
        echo
        exit 1
    ;;
esac

case "$SINGULARITY_IMAGE" in
    *.cpioz|*.vnfs)
        NAME=`basename "$SINGULARITY_IMAGE"`
        TIMESTAMP=`stat -c "%Y" "$SINGULARITY_IMAGE"`
        if [ -z "${SINGULARITY_CACHEDIR:-}" ]; then
            USERID=`id -u`
            HOMEDIR=`getent passwd $USERID | cut -d: -f6`
            SINGULARITY_CACHEDIR="$HOMEDIR/.singularity/cache"
        fi
        CONTAINER_DIR="$SINGULARITY_CACHEDIR/$NAME/$TIMESTAMP/$NAME"
        if [ ! -d "$CONTAINER_DIR" ]; then
            if ! mkdir -p "$CONTAINER_DIR"; then
                message ERROR "Could not create cache directory: $CONTAINER_DIR\n"
                ABORT 255
            fi
            message 1 "Opening cpio archive, stand by...\n"
            # this almost always gives permission errors, so ignore them when
            # running as a user.
            zcat "$SINGULARITY_IMAGE" | ( cd "$CONTAINER_DIR"; cpio -id >/dev/null 2>&1 )
        fi
        SINGULARITY_IMAGE="$CONTAINER_DIR"
    ;;
    *.cpio)
        NAME=`basename "$SINGULARITY_IMAGE"`
        TIMESTAMP=`stat -c "%Y" "$SINGULARITY_IMAGE"`
        if [ -z "${SINGULARITY_CACHEDIR:-}" ]; then
            USERID=`id -u`
            HOMEDIR=`getent passwd $USERID | cut -d: -f6`
            SINGULARITY_CACHEDIR="$HOMEDIR/.singularity/cache"
        fi
        CONTAINER_DIR="$SINGULARITY_CACHEDIR/$NAME/$TIMESTAMP/$NAME"
        if [ ! -d "$CONTAINER_DIR" ]; then
            if ! mkdir -p "$CONTAINER_DIR"; then
                message ERROR "Could not create cache directory: $CONTAINER_DIR\n"
                ABORT 255
            fi
            message 1 "Opening cpio archive, stand by...\n"
            # this almost always gives permission errors, so ignore them when
            # running as a user.
            cat "$SINGULARITY_IMAGE" | ( cd "$CONTAINER_DIR"; cpio -id >/dev/null 2>&1 )
        fi
        SINGULARITY_IMAGE="$CONTAINER_DIR"
    ;;
    *.tar)
        NAME=`basename "$SINGULARITY_IMAGE"`
        TIMESTAMP=`stat -c "%Y" "$SINGULARITY_IMAGE"`
        if [ -z "${SINGULARITY_CACHEDIR:-}" ]; then
            USERID=`id -u`
            HOMEDIR=`getent passwd $USERID | cut -d: -f6`
            SINGULARITY_CACHEDIR="$HOMEDIR/.singularity/cache"
        fi
        CONTAINER_DIR="$SINGULARITY_CACHEDIR/$NAME/$TIMESTAMP/$NAME"
        if [ ! -d "$CONTAINER_DIR" ]; then
            if ! mkdir -p "$CONTAINER_DIR"; then
                message ERROR "Could not create cache directory: $CONTAINER_DIR\n"
                ABORT 255
            fi
            message 1 "Opening tar archive, stand by...\n"
            # this almost always gives permission errors, so ignore them when
            # running as a user.
            tar -C "$CONTAINER_DIR" -xf "$SINGULARITY_IMAGE" 2>/dev/null
        fi
        SINGULARITY_IMAGE="$CONTAINER_DIR"
    ;;
    *.tgz|*.tar.gz)
        NAME=`basename "$SINGULARITY_IMAGE"`
        TIMESTAMP=`stat -c "%Y" "$SINGULARITY_IMAGE"`
        if [ -z "${SINGULARITY_CACHEDIR:-}" ]; then
            USERID=`id -u`
            HOMEDIR=`getent passwd $USERID | cut -d: -f6`
            SINGULARITY_CACHEDIR="$HOMEDIR/.singularity/cache"
        fi
        CONTAINER_DIR="$SINGULARITY_CACHEDIR/$NAME/$TIMESTAMP/$NAME"
        if [ ! -d "$CONTAINER_DIR" ]; then
            if ! mkdir -p "$CONTAINER_DIR"; then
                message ERROR "Could not create cache directory: $CONTAINER_DIR\n"
                ABORT 255
            fi
            message 1 "Opening gzip compressed archive, stand by...\n"
            # this almost always gives permission errors, so ignore them when
            # this almost always gives permission errors, so ignore them when
            # running as a user.
            tar -C "$CONTAINER_DIR" -xzf "$SINGULARITY_IMAGE"
        fi
        SINGULARITY_IMAGE="$CONTAINER_DIR"
    ;;
    *.tbz|*.tar.bz)
        NAME=`basename "$SINGULARITY_IMAGE"`
        TIMESTAMP=`stat -c "%Y" "$SINGULARITY_IMAGE"`
        if [ -z "${SINGULARITY_CACHEDIR:-}" ]; then
            USERID=`id -u`
            HOMEDIR=`getent passwd $USERID | cut -d: -f6`
            SINGULARITY_CACHEDIR="$HOMEDIR/.singularity/cache"
        fi
        CONTAINER_DIR="$SINGULARITY_CACHEDIR/$NAME/$TIMESTAMP/$NAME"
        if [ ! -d "$CONTAINER_DIR" ]; then
            if ! mkdir -p "$CONTAINER_DIR"; then
                message ERROR "Could not create cache directory: $CONTAINER_DIR\n"
                ABORT 255
            fi
            message 1 "Opening bzip compressed archive, stand by...\n"
            # this almost always gives permission errors, so ignore them when
            # running as a user.
            tar -C "$CONTAINER_DIR" -xjf "$SINGULARITY_IMAGE" 2>/dev/null
        fi
        SINGULARITY_IMAGE="$CONTAINER_DIR"
    ;;
esac

