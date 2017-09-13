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


eval_abort "$SINGULARITY_libexecdir/singularity/bootstrap-scripts/pre.sh"
eval_abort "$SINGULARITY_libexecdir/singularity/bootstrap-scripts/environment.sh"

if [ -n "${BOOTSTRAP:-}" -a -z "${SINGULARITY_BUILDNOBASE:-}" ]; then
    if [ -x "$SINGULARITY_libexecdir/singularity/bootstrap-scripts/deffile-driver-$BOOTSTRAP.sh" ]; then
        if [ ! -f "${SINGULARITY_ROOTFS}/.coredone" ]; then
            eval_abort "$SINGULARITY_libexecdir/singularity/bootstrap-scripts/deffile-driver-$BOOTSTRAP.sh"
            touch "${SINGULARITY_ROOTFS}/.coredone"
        fi
    else
        message ERROR "'Bootstrap' type not supported: $BOOTSTRAP\n"
        exit 1
    fi
fi

# take a snapshot of the environment for later comparison
if [ "${BOOTSTRAP:-}" = "localimage" -o "${BOOTSTRAP:-}" = "shub" ]; then
    SINGULARITY_STARTING_ENVIRONMENT=$(eval_abort env -i ${SINGULARITY_libexecdir}/singularity/helpers/record-env.sh ${SINGULARITY_ROOTFS})
    SINGULARITY_STARTING_ENVSHA1=$(eval_abort sha1sum ${SINGULARITY_ROOTFS}/.singularity.d/env/*.sh | sha1sum)
    export SINGULARITY_STARTING_ENVIRONMENT SINGULARITY_STARTING_ENVSHA1
fi 

eval_abort "$SINGULARITY_libexecdir/singularity/bootstrap-scripts/deffile-sections.sh"
eval_abort "$SINGULARITY_libexecdir/singularity/bootstrap-scripts/post.sh"

# take another snapshot and compare to see what changed
if [ "${BOOTSTRAP:-}" = "localimage" -o "${BOOTSTRAP:-}" = "shub" ]; then
    SINGULARITY_ENDING_ENVIRONMENT=$(eval_abort env -i ${SINGULARITY_libexecdir}/singularity/helpers/record-env.sh ${SINGULARITY_ROOTFS})
    SINGULARITY_ENDING_ENVSHA1=$(eval_abort sha1sum ${SINGULARITY_ROOTFS}/.singularity.d/env/*.sh | sha1sum)
    export SINGULARITY_ENDING_ENVIRONMENT SINGULARITY_ENDING_ENVSHA1
    compare_envs
    rm $SINGULARITY_STARTING_ENVIRONMENT $SINGULARITY_ENDING_ENVIRONMENT
fi 

# If checks specified, export variable
if [ "${SINGULARITY_CHECKS:-}" = "no" ]; then
    message 1 "Skipping checks\n"
else
    message 1 "Running checks\n"
    eval_abort "$SINGULARITY_libexecdir/singularity/bootstrap-scripts/checks.sh"
fi
