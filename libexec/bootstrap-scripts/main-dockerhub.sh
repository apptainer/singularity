#!/bin/bash
#
# Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
#
# See the COPYRIGHT.md file at the top-level directory of this distribution and at
# https://github.com/singularityware/singularity/blob/master/COPYRIGHT.md.
#
# This file is part of the Singularity Linux container project. It is subject to the license
# terms in the LICENSE.md file found in the top-level directory of this distribution and
# at https://github.com/singularityware/singularity/blob/master/LICENSE.md. No part
# of Singularity, including this file, may be copied, modified, propagated, or distributed
# except according to the terms contained in the LICENSE.md file.
#
# This file also contains content that is covered under the LBNL/DOE/UC modified
# 3-clause BSD license and is subject to the license terms in the LICENSE-LBNL.md
# file found in the top-level directory of this distribution and at
# https://github.com/singularityware/singularity/blob/master/LICENSE-LBNL.md.


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


SINGULARITY_CONTAINER="$SINGULARITY_BUILDDEF"
if ! SINGULARITY_CONTENTS=`mktemp ${TMPDIR:-/tmp}/.singularity-layers.XXXXXXXX`; then
    message ERROR "Failed to create temporary directory\n"
    ABORT 255
fi
export SINGULARITY_CONTAINER SINGULARITY_CONTENTS

eval_abort "$SINGULARITY_libexecdir/singularity/bootstrap-scripts/pre.sh"
eval_abort "$SINGULARITY_libexecdir/singularity/bootstrap-scripts/environment.sh"
eval_abort "$SINGULARITY_libexecdir/singularity/python/import.py"

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
    message 1 "Exploding layer: $name\n"
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
    zcat "$i" | (cd "$SINGULARITY_ROOTFS"; tar --exclude=dev/* --exclude=*/.wh..wh..opq -xf - ) || exit $?
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

# If checktags not defined, default to docker
if [ -z "${SINGULARITY_CHECKTAGS:-}" ]; then
    SINGULARITY_CHECKTAGS=docker
    export SINGULARITY_CHECKTAGS
fi


eval_abort "$SINGULARITY_libexecdir/singularity/bootstrap-scripts/post.sh"

# If checks specified, export variable
if [ "${SINGULARITY_CHECKS:-}" = "no" ]; then
    message 1 "Skipping checks\n"
else
    message 1 "Running checks\n"
    eval_abort "$SINGULARITY_libexecdir/singularity/bootstrap-scripts/checks.sh"
fi
