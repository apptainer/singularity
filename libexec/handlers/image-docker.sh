#!/bin/bash
# 
# Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
# Copyright (c) 2015-2017, Gregory M. Kurtzer. All rights reserved.


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

# Docker layers are extracted with tar, but we need additional handling
# for aufs whiteout files that may be present.
# - .wh..whi..opq inside a directory indicates that directory is opaque.
#    Any content in this directory from previous layers must be deleted
#    before extracting the current layer.
# - .wh.<file/dirname> indicates <file/dirname> from a previous layer must
#    be deleted as we extract the current layer.
# See:
#   https://github.com/tonistiigi/docker/blob/2fb5d0c32376951ef41a6f64bb7dbd8f6fd14fba/pkg/archive/whiteouts.go#L3-L2
for i in `cat "$SINGULARITY_CONTENTS"`; do
    name=`basename "$i"`
    message 2 "Exploding layer: $name\n"

    # Remove any existing directories containing a whiteout opaque
    # indicator file .wh..wh..opq in the layer we are about to extract
    # When we extract the tar they will be re-created, and empty
    # (opaque) hiding anything from earlier layers.
    zcat "$i" | tar --quoting-style=escape -tf - | grep '\.wh\.\.wh\.\.opq' | while read OPQ
    do
        OPAQUE_DIR=$(dirname "$OPQ")
        if [ -d "${SINGULARITY_ROOTFS}${OPAQUE_DIR}" ]; then
            message 2 "Making $OPAQUE_DIR opaque\n"
            rm -rf "${SINGULARITY_ROOTFS}${OPAQUE_DIR}"
        fi
    done

    # Now extract our current layer, exclude whiteout opaque marker handled
    # above so they don't interfere in the logic below.
    ( zcat "$i" | (cd "$SINGULARITY_ROOTFS"; tar --overwrite --exclude=dev/* --exclude=*/.wh.* -xvf -) || exit $? ) | while read file; do
        if [ -L "$SINGULARITY_ROOTFS/$file" ]; then
            # Skipping symlinks
            true
        elif [ -f "$SINGULARITY_ROOTFS/$file" ]; then
            chmod u+rw "$SINGULARITY_ROOTFS/$file" >/dev/null 2>&1
        elif [ -d "$SINGULARITY_ROOTFS/$file" ]; then
            chmod u+rwx "$SINGULARITY_ROOTFS/${file%/}" >/dev/null 2>&1
        fi
    done

    # Get rid of any files/dirs marked for white out
    find "$SINGULARITY_ROOTFS" -type f -name '.wh.*' -print | while read WHITEOUT
    do
        WHITEOUT_TARGET=${WHITEOUT/\/\.wh\.//}
        message 2 "Removing whiteout-ed file/dir $WHITEOUT_TARGET\n"
        rm -rf "$WHITEOUT_TARGET"
    done
    # Get rid of all of the whiteout indicator files themselves
    find "$SINGULARITY_ROOTFS" -type f -name '.wh.*' -delete

done

rm -f "$SINGULARITY_CONTENTS"
