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
#    be deleted before we extract the current layer.
# See:
#   https://github.com/tonistiigi/docker/blob/2fb5d0c32376951ef41a6f64bb7dbd8f6fd14fba/pkg/archive/whiteouts.go#L3-L23
for i in `cat "$SINGULARITY_CONTENTS"`; do
    message 1 "Importing: $i\n"
    zcat "$i" | tar --quoting-style=escape -tf - | grep '\.wh\.' | while read WHITEOUT
    do
        # Handle opaque directories (remove them, will be recreated
        # empty at extraction of tar).
        if [[ $WHITEOUT == *.wh..wh..opq ]]; then
            OPAQUE_DIR=$(dirname "$WHITEOUT")
            if [ -d "${SINGULARITY_ROOTFS}${OPAQUE_DIR}" ]; then
                message 2 "Making $OPAQUE_DIR opaque\n"
                rm -rf "${SINGULARITY_ROOTFS}${OPAQUE_DIR}"
            fi
        # Handle other plain whiteout marker files. Remove the target
        # before extraction.
        else
            WHITEOUT_TARGET="${WHITEOUT/\/\.wh\.//}"
            message 2 "Removing whiteout-ed file/dir $WHITEOUT_TARGET\n"
            rm -rf "${SINGULARITY_ROOTFS}/${WHITEOUT_TARGET}"
        fi
    done
    # Now extract our current layer, exclude whiteout files handled
    # above.
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
done

rm -f "$SINGULARITY_CONTENTS"
