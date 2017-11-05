#!/bin/bash
#
# Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
# Copyright (c) 2017, Vanessa Sochat. All rights reserved.
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

if [ -z "${FROM:-}" ]; then
    message ERROR "Required Definition tag 'From:' not defined.\n"
    exit 1
fi

################################################################################
# Docker Customizations
################################################################################

if [ ! -z "${REGISTRY:-}" ]; then
    message DEBUG "Custom Docker Registry 'Registry:' ${REGISTRY}.\n"
    export REGISTRY
fi

# Note: NAMESPACE can be set to an empty string, and that's a valid namespace
# for Docker (not so for shub://)
if [ -n "${NAMESPACE+set}" ]; then
    message DEBUG "Custom Docker Namespace 'Namespace:' ${NAMESPACE}.\n"
    export NAMESPACE
fi

if [ -z "${INCLUDECMD:-}" ]; then
    export SINGULARITY_INCLUDECMD="yes"
fi



SINGULARITY_CONTAINER="docker://$FROM"
SINGULARITY_LABELFILE="$SINGULARITY_ROOTFS/.singularity.d/labels.json"

if ! SINGULARITY_CONTENTS=`mktemp ${TMPDIR:-/tmp}/.singularity-layers.XXXXXXXX`; then
    message ERROR "Failed to create temporary directory\n"
    ABORT 255
fi
export SINGULARITY_CONTAINER SINGULARITY_CONTENTS SINGULARITY_LABELFILE

eval_abort "$SINGULARITY_libexecdir/singularity/python/import.py"

umask 0002
for i in `cat "$SINGULARITY_CONTENTS"`; do
    name=`basename "$i"`
    message 1 "Exploding layer: $name\n"
    zcat "$i" | (cd "$SINGULARITY_ROOTFS"; tar --exclude=dev/* -xf -) || exit $?
done

rm -f "$SINGULARITY_CONTENTS"


exit 0
