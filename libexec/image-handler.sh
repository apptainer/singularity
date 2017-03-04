#!/bin/bash
# 
# Copyright (c) 2015-2017, Gregory M. Kurtzer. All rights reserved.
# 
# Copyright (c) 2016-2017, The Regents of the University of California,
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
        NAME=`echo "$SINGULARITY_IMAGE" | sed -e 's@^docker://@@'`
        if [ -n "${SINGULARITY_CACHEDIR:-}" ]; then
            SINGULARITY_CACHEDIR_LOCAL="$SINGULARITY_CACHEDIR"
        else
            SINGULARITY_CACHEDIR_LOCAL="/tmp"
        fi

        if ! SINGULARITY_SESSIONDIR=`mktemp -d $SINGULARITY_CACHEDIR_LOCAL/.singularity-runtime.XXXXXXXX`; then
            message ERROR "Failed to create cleandir\n"
            ABORT 255
        fi

        SINGULARITY_ROOTFS="$SINGULARITY_SESSIONDIR/container/$NAME"
        if ! mkdir -p "$SINGULARITY_ROOTFS"; then
            message ERROR "Failed to create named SINGULARITY_ROOTFS=$SINGULARITY_ROOTFS\n"
            ABORT 255
        fi

        SINGULARITY_CONTAINER="$SINGULARITY_IMAGE"
        SINGULARITY_IMAGE="$SINGULARITY_ROOTFS"

        export SINGULARITY_ROOTFS SINGULARITY_IMAGE SINGULARITY_CONTAINER SINGULARITY_SESSIONDIR

        zcat $SINGULARITY_libexecdir/singularity/bootstrap-scripts/environment.tar | (cd $SINGULARITY_ROOTFS; tar -xf -) || exit $?


        if ! eval "$SINGULARITY_libexecdir/singularity/python/docker/import.py"; then
            ABORT $?
        fi

        chmod -R +w "$SINGULARITY_ROOTFS"

    ;;
    shub://*)
        NAME=`echo "$SINGULARITY_IMAGE" | sed -e 's@^shub://@@'`

        if [ -n "${SINGULARITY_CACHEDIR:-}" ]; then
            SINGULARITY_CACHEDIR_LOCAL="$SINGULARITY_CACHEDIR"
        else
            SINGULARITY_CACHEDIR_LOCAL="/tmp"
        fi
        if ! BASE_CONTAINER_DIR=`mktemp -d $SINGULARITY_CACHEDIR_LOCAL/singularity-container_dir.XXXXXXXX`; then
            message ERROR "Failed to create container_dir\n"
            ABORT 255
        fi

        CONTAINER_DIR="$BASE_CONTAINER_DIR/$NAME"
        if ! mkdir -p "$CONTAINER_DIR"; then
            message ERROR "Failed to create named container_dir\n"
            ABORT 255
        fi

        eval $SINGULARITY_libexecdir/singularity/python/cli.py --shub $NAME --rootfs $CONTAINER_DIR

        # The python script saves names to files in CONTAINER_DIR
        SINGULARITY_IMAGE=`cat $CONTAINER_DIR/SINGULARITY_IMAGE`
        export SINGULARITY_IMAGE

    ;;
esac

case "$SINGULARITY_IMAGE" in
    *.cpioz|*.vnfs)
        NAME=`basename "$SINGULARITY_IMAGE"`
        if [ -z "${SINGULARITY_CACHEDIR:-}" ]; then
            SINGULARITY_CACHEDIR="/tmp"
        fi
        if [ ! -d "$SINGULARITY_CACHEDIR" ]; then
            message ERROR "Cache directory does not exist: $SINGULARITY_CACHEDIR\n"
            ABORT 1
        fi
        if ! SINGULARITY_TMPDIR=`mktemp -d $SINGULARITY_CACHEDIR/singularity-rundir.XXXXXXXX`; then
            message ERROR "Failed to create tmpdir\n"
            ABORT 255
        fi

        CONTAINER_DIR="$SINGULARITY_TMPDIR/$NAME"
        if ! mkdir -p "$CONTAINER_DIR"; then
            message ERROR "Could not create cache directory: $CONTAINER_DIR\n"
            ABORT 255
        fi

        message 1 "Opening cpio archive, stand by...\n"
        # this almost always gives permission errors, so ignore them when
        # running as a user.
        zcat "$SINGULARITY_IMAGE" | ( cd "$CONTAINER_DIR"; cpio -id >/dev/null 2>&1 )

        chmod -R +w "$CONTAINER_DIR"

        SINGULARITY_IMAGE="$CONTAINER_DIR"
        SINGULARITY_RUNDIR="$SINGULARITY_TMPDIR"
        export SINGULARITY_RUNDIR SINGULARITY_IMAGE
    ;;
    *.cpio)
        NAME=`basename "$SINGULARITY_IMAGE"`
        if [ -z "${SINGULARITY_CACHEDIR:-}" ]; then
            SINGULARITY_CACHEDIR="/tmp"
        fi
        if [ ! -d "$SINGULARITY_CACHEDIR" ]; then
            message ERROR "Cache directory does not exist: $SINGULARITY_CACHEDIR\n"
            ABORT 1
        fi
        if ! SINGULARITY_TMPDIR=`mktemp -d $SINGULARITY_CACHEDIR/singularity-rundir.XXXXXXXX`; then
            message ERROR "Failed to create tmpdir\n"
            ABORT 255
        fi

        CONTAINER_DIR="$SINGULARITY_TMPDIR/$NAME"
        if ! mkdir -p "$CONTAINER_DIR"; then
            message ERROR "Could not create cache directory: $CONTAINER_DIR\n"
            ABORT 255
        fi

        message 1 "Opening cpio archive, stand by...\n"
        # this almost always gives permission errors, so ignore them when
        # running as a user.
        cat "$SINGULARITY_IMAGE" | ( cd "$CONTAINER_DIR"; cpio -id >/dev/null 2>&1 )

        chmod -R +w "$CONTAINER_DIR"

        SINGULARITY_IMAGE="$CONTAINER_DIR"
        SINGULARITY_RUNDIR="$SINGULARITY_TMPDIR"
        export SINGULARITY_RUNDIR SINGULARITY_IMAGE
    ;;
    *.tar)
        NAME=`basename "$SINGULARITY_IMAGE"`
        if [ -z "${SINGULARITY_CACHEDIR:-}" ]; then
            SINGULARITY_CACHEDIR="/tmp"
        fi
        if [ ! -d "$SINGULARITY_CACHEDIR" ]; then
            message ERROR "Cache directory does not exist: $SINGULARITY_CACHEDIR\n"
            ABORT 1
        fi
        if ! SINGULARITY_TMPDIR=`mktemp -d $SINGULARITY_CACHEDIR/singularity-rundir.XXXXXXXX`; then
            message ERROR "Failed to create tmpdir\n"
            ABORT 255
        fi

        CONTAINER_DIR="$SINGULARITY_TMPDIR/$NAME"
        if ! mkdir -p "$CONTAINER_DIR"; then
            message ERROR "Could not create cache directory: $CONTAINER_DIR\n"
            ABORT 255
        fi

        message 1 "Opening tar archive, stand by...\n"
        # this almost always gives permission errors, so ignore them when
        # running as a user.
        tar -C "$CONTAINER_DIR" -xf "$SINGULARITY_IMAGE" 2>/dev/null

        chmod -R +w "$CONTAINER_DIR"

        SINGULARITY_IMAGE="$CONTAINER_DIR"
        SINGULARITY_RUNDIR="$SINGULARITY_TMPDIR"
        export SINGULARITY_RUNDIR SINGULARITY_IMAGE
    ;;
    *.tgz|*.tar.gz)
        NAME=`basename "$SINGULARITY_IMAGE"`
        if [ -z "${SINGULARITY_CACHEDIR:-}" ]; then
            SINGULARITY_CACHEDIR="/tmp"
        fi
        if [ ! -d "$SINGULARITY_CACHEDIR" ]; then
            message ERROR "Cache directory does not exist: $SINGULARITY_CACHEDIR\n"
            ABORT 1
        fi
        if ! SINGULARITY_TMPDIR=`mktemp -d $SINGULARITY_CACHEDIR/singularity-rundir.XXXXXXXX`; then
            message ERROR "Failed to create tmpdir\n"
            ABORT 255
        fi

        CONTAINER_DIR="$SINGULARITY_TMPDIR/$NAME"
        if ! mkdir -p "$CONTAINER_DIR"; then
            message ERROR "Could not create cache directory: $CONTAINER_DIR\n"
            ABORT 255
        fi

        message 1 "Opening gzip compressed archive, stand by...\n"
        # this almost always gives permission errors, so ignore them when
        # this almost always gives permission errors, so ignore them when
        # running as a user.
        tar -C "$CONTAINER_DIR" -xzf "$SINGULARITY_IMAGE" 2>/dev/null

        chmod -R +w "$CONTAINER_DIR"

        SINGULARITY_IMAGE="$CONTAINER_DIR"
        SINGULARITY_RUNDIR="$SINGULARITY_TMPDIR"
        export SINGULARITY_RUNDIR SINGULARITY_IMAGE
    ;;
    *.tbz|*.tar.bz)
        NAME=`basename "$SINGULARITY_IMAGE"`
        if [ -z "${SINGULARITY_CACHEDIR:-}" ]; then
            SINGULARITY_CACHEDIR="/tmp"
        fi
        if [ ! -d "$SINGULARITY_CACHEDIR" ]; then
            message ERROR "Cache directory does not exist: $SINGULARITY_CACHEDIR\n"
            ABORT 1
        fi
        if ! SINGULARITY_TMPDIR=`mktemp -d $SINGULARITY_CACHEDIR/singularity-rundir.XXXXXXXX`; then
            message ERROR "Failed to create tmpdir\n"
            ABORT 255
        fi

        CONTAINER_DIR="$SINGULARITY_TMPDIR/$NAME"
        if ! mkdir -p "$CONTAINER_DIR"; then
            message ERROR "Could not create cache directory: $CONTAINER_DIR\n"
            ABORT 255
        fi

        message 1 "Opening bzip compressed archive, stand by...\n"
        # this almost always gives permission errors, so ignore them when
        # running as a user.
        tar -C "$CONTAINER_DIR" -xjf "$SINGULARITY_IMAGE" 2>/dev/null

        chmod -R +w "$CONTAINER_DIR"

        SINGULARITY_IMAGE="$CONTAINER_DIR"
        SINGULARITY_RUNDIR="$SINGULARITY_TMPDIR"
        export SINGULARITY_RUNDIR SINGULARITY_IMAGE
    ;;
esac

