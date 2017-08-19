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
        eval_abort "$SINGULARITY_libexecdir/singularity/bootstrap-scripts/deffile-driver-$BOOTSTRAP.sh"
    else
        message ERROR "'Bootstrap' type not supported: $BOOTSTRAP\n"
        exit 1
    fi
fi

# take a snapshot of the environment for later comparison
if [ ${BOOTSTRAP:-} = "localimage" ]; then
    SINGULARITY_STARTING_ENVIRONMENT=$(eval_abort env -i ${SINGULARITY_libexecdir}/singularity/helpers/record-env.sh ${SINGULARITY_ROOTFS})
    SINGULARITY_STARTING_ENVSHA1=$(eval_abort sha1sum ${SINGULARITY_ROOTFS}/.singularity.d/env/*.sh | sha1sum)
    export SINGULARITY_STARTING_ENVIRONMENT SINGULARITY_STARTING_ENVSHA1
fi 

eval_abort "$SINGULARITY_libexecdir/singularity/bootstrap-scripts/deffile-sections.sh"
eval_abort "$SINGULARITY_libexecdir/singularity/bootstrap-scripts/post.sh"

# take another snapshot and compare to see what changed
if [ ${BOOTSTRAP:-} = "localimage" ]; then
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
