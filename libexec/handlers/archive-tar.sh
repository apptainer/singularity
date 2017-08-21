#!/bin/bash
# 
# Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
# Copyright (c) 2015-2017, Gregory M. Kurtzer. All rights reserved.


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


case "$SINGULARITY_IMAGE" in

    *.tar)

        message 1 "Opening tar archive, stand by...\n"
        # this almost always gives permission errors, so ignore them when
        # running as a user.
        tar -C "$CONTAINER_DIR" -xf "$SINGULARITY_IMAGE" 2>/dev/null

    ;;
    *.tgz|*.tar.gz)

        message 1 "Opening gzip compressed archive, stand by...\n"

        # this almost always gives permission errors, so ignore them when
        # running as a user.
        tar -C "$CONTAINER_DIR" -xzf "$SINGULARITY_IMAGE" 2>/dev/null

    ;;
    *.tbz|*.tar.bz)

        message 1 "Opening bzip compressed archive, stand by...\n"
        # this almost always gives permission errors, so ignore them when
        # running as a user.
        tar -C "$CONTAINER_DIR" -xjf "$SINGULARITY_IMAGE" 2>/dev/null

    ;;
esac

chmod -R +w "$CONTAINER_DIR"

SINGULARITY_IMAGE="$CONTAINER_DIR"
SINGULARITY_CLEANUPDIR="$SINGULARITY_TMPDIR"
export SINGULARITY_CLEANUPDIR SINGULARITY_IMAGE
