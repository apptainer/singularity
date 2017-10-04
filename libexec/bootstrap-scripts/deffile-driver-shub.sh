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
    message ERROR "Singularity root file system not defined\n"
    exit 1
fi

if [ -z "${FROM:-}" ]; then
    message ERROR "Required Definition tag 'From:' not defined.\n"
    exit 1
fi


################################################################################
# Singularity Hub/Registry Customizations
################################################################################

if [ ! -z "${REGISTRY:-}" ]; then
    message DEBUG "Custom Singularity Registry 'Registry:' ${REGISTRY}.\n"
    export REGISTRY
fi

if [ ! -z "${NAMESPACE:-}" ]; then
    message DEBUG "Custom Singularity Registry Namespace 'Namespace:' ${NAMESPACE}.\n"
    export NAMESPACE
fi


########## BEGIN BOOTSTRAP SCRIPT ##########
SINGULARITY_CONTAINER="shub://${FROM}"
if ! SINGULARITY_CONTENTS=`mktemp ${TMPDIR:-/tmp}/.singularity-layerfile.XXXXXX`; then
    message ERROR "Failed to create temporary directory\n"
    ABORT 255
fi
        
# If cache is set, set pull folder to it (first priority)
if [ -n "${SINGULARITY_CACHEDIR:-}" ]; then
    SINGULARITY_PULLFOLDER="$SINGULARITY_CACHEDIR"
else
    # Only set the pull folder to be $PWD if not set by user
    if [ ! -n "${SINGULARITY_PULLFOLDER:-}" ]; then
        SINGULARITY_PULLFOLDER="."
    fi
fi

export SINGULARITY_CONTENTS SINGULARITY_CONTAINER SINGULARITY_PULLFOLDER

umask 0002

# relying on pull.py for error checking here
${SINGULARITY_libexecdir}/singularity/python/pull.py

message 1 "Exporting contents of ${SINGULARITY_CONTAINER} to ${SINGULARITY_IMAGE}\n"

# switch $SINGULARITY_CONTAINER from remote to local 
SINGULARITY_CONTAINER=`cat $SINGULARITY_CONTENTS`
rm -r $SINGULARITY_CONTENTS

#if ! eval "${SINGULARITY_bindir}"/singularity image.export "${SINGULARITY_CONTAINER}" | (cd "${SINGULARITY_ROOTFS}" && tar xBf -); then
if ! eval "${SINGULARITY_bindir}"/singularity image.export "${SINGULARITY_CONTAINER}" | tar xBf - -C "${SINGULARITY_ROOTFS}"; then
    message ERROR "Failed to export contents of ${SINGULARITY_CONTAINER} to ${SINGULARITY_ROOTFS}\n"
    rm $SINGULARITY_CONTAINER
    ABORT 255
fi
rm $SINGULARITY_CONTAINER
