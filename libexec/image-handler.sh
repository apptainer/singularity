#!/bin/bash
# 
# Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
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

        if [ -z "${SINGULARITY_LOCALCACHEDIR:-}" ]; then
            SINGULARITY_LOCALCACHEDIR="${TMPDIR:-/tmp}"
        fi

        if ! SINGULARITY_TMPDIR=`mktemp -d $SINGULARITY_LOCALCACHEDIR/.singularity-runtime.XXXXXXXX`; then
            message ERROR "Failed to create temporary directory\n"
            ABORT 255
        fi

        SINGULARITY_ROOTFS="$SINGULARITY_TMPDIR/$NAME"
        if ! mkdir -p "$SINGULARITY_ROOTFS"; then
            message ERROR "Failed to create named SINGULARITY_ROOTFS=$SINGULARITY_ROOTFS\n"
            ABORT 255
        fi

        SINGULARITY_CONTAINER="$SINGULARITY_IMAGE"
        SINGULARITY_IMAGE="$SINGULARITY_ROOTFS"
        SINGULARITY_CLEANUPDIR="$SINGULARITY_TMPDIR"
        if ! SINGULARITY_CONTENTS=`mktemp ${TMPDIR:-/tmp}/.singularity-layers.XXXXXXXX`; then
            message ERROR "Failed to create temporary directory\n"
            ABORT 255
        fi

        export SINGULARITY_ROOTFS SINGULARITY_IMAGE SINGULARITY_CONTAINER SINGULARITY_CONTENTS SINGULARITY_CLEANUPDIR

        eval_abort "$SINGULARITY_libexecdir/singularity/python/import.py"

        message 1 "Creating container runtime...\n"
        message 2 "Importing: base Singularity environment\n"
        zcat $SINGULARITY_libexecdir/singularity/bootstrap-scripts/environment.tar | (cd $SINGULARITY_ROOTFS; tar -xf -) || exit $?
         
        for i in `cat "$SINGULARITY_CONTENTS"`; do
            name=`basename "$i"`
            message 2 "Exploding layer: $name\n"
            ( zcat "$i" | (cd "$SINGULARITY_ROOTFS"; tar --overwrite --exclude=dev/* -xvf -) || exit $? ) | while read file; do
                if [ -L "$SINGULARITY_ROOTFS/$file" ]; then
                    # Skipping symlinks
                    true
                elif [ -f "$SINGULARITY_ROOTFS/$file" ]; then
                    chmod u+rw "$SINGULARITY_ROOTFS/$file"
                elif [ -d "$SINGULARITY_ROOTFS/$file" ]; then
                    chmod u+rwx "$SINGULARITY_ROOTFS/${file%/}"
                fi
            done
        done

        rm -f "$SINGULARITY_CONTENTS"

    ;;
    shub://*)
        if ! SINGULARITY_CONTENTS=`mktemp ${TMPDIR:-/tmp}/.singularity-layerfile.XXXXXX`; then
            message ERROR "Failed to create temporary directory\n"
            ABORT 255
        fi

        if [ -n "${SINGULARITY_CACHEDIR:-}" ]; then
            SINGULARITY_PULLFOLDER="$SINGULARITY_CACHEDIR"
        else
            SINGULARITY_PULLFOLDER="."
        fi

        SINGULARITY_CONTAINER="$SINGULARITY_IMAGE"
        export SINGULARITY_PULLFOLDER SINGULARITY_CONTAINER SINGULARITY_CONTENTS

        if ! eval "$SINGULARITY_libexecdir/singularity/python/pull.py"; then
            ABORT 255
        fi

        # The python script saves names to files in CONTAINER_DIR
        SINGULARITY_IMAGE=`cat $SINGULARITY_CONTENTS`
        export SINGULARITY_IMAGE

        rm -f "$SINGULARITY_CONTENTS"

        if [ -f "$SINGULARITY_IMAGE" ]; then
            chmod +x "$SINGULARITY_IMAGE"
        else
            message ERROR "Could not locate downloaded image\n"
            ABORT 255
        fi
    ;;
esac

case "$SINGULARITY_IMAGE" in
    *.cpioz|*.vnfs)
        NAME=`basename "$SINGULARITY_IMAGE"`
        if [ -z "${SINGULARITY_CACHEDIR:-}" ]; then
            SINGULARITY_CACHEDIR="${TMPDIR:-/tmp}"
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
        SINGULARITY_CLEANUPDIR="$SINGULARITY_TMPDIR"
        export SINGULARITY_CLEANUPDIR SINGULARITY_IMAGE
    ;;
    *.cpio)
        NAME=`basename "$SINGULARITY_IMAGE"`
        if [ -z "${SINGULARITY_CACHEDIR:-}" ]; then
            SINGULARITY_CACHEDIR="${TMPDIR:-/tmp}"
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
        SINGULARITY_CLEANUPDIR="$SINGULARITY_TMPDIR"
        export SINGULARITY_CLEANUPDIR SINGULARITY_IMAGE
    ;;
    *.tar)
        NAME=`basename "$SINGULARITY_IMAGE"`
        if [ -z "${SINGULARITY_CACHEDIR:-}" ]; then
            SINGULARITY_CACHEDIR="${TMPDIR:-/tmp}"
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
        SINGULARITY_CLEANUPDIR="$SINGULARITY_TMPDIR"
        export SINGULARITY_CLEANUPDIR SINGULARITY_IMAGE
    ;;
    *.tgz|*.tar.gz)
        NAME=`basename "$SINGULARITY_IMAGE"`
        if [ -z "${SINGULARITY_CACHEDIR:-}" ]; then
            SINGULARITY_CACHEDIR="${TMPDIR:-/tmp}"
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
        SINGULARITY_CLEANUPDIR="$SINGULARITY_TMPDIR"
        export SINGULARITY_CLEANUPDIR SINGULARITY_IMAGE
    ;;
    *.tbz|*.tar.bz)
        NAME=`basename "$SINGULARITY_IMAGE"`
        if [ -z "${SINGULARITY_CACHEDIR:-}" ]; then
            SINGULARITY_CACHEDIR="${TMPDIR:-/tmp}"
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
        SINGULARITY_CLEANUPDIR="$SINGULARITY_TMPDIR"
        export SINGULARITY_CLEANUPDIR SINGULARITY_IMAGE
    ;;
esac

