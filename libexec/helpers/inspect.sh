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

if [ ! -d "${SINGULARITY_MOUNTPOINT}" ]; then
    message ERROR "The mount point does not exist: ${SINGULARITY_MOUNTPOINT}\n"
    ABORT 255
fi

if [ ! -d "${SINGULARITY_MOUNTPOINT}/.singularity.d" ]; then
    message ERROR "The Singularity metadata directory does not exist in image\n"
    ABORT 255
fi


if [ -n "${SINGULARITY_INSPECT_LABELS:-}" ]; then
    if [ -f "$SINGULARITY_MOUNTPOINT/.singularity.d/labels.json" ]; then
        message 1 "## LABELS:\n"
        cat "$SINGULARITY_MOUNTPOINT/.singularity.d/labels.json"
        echo
    else
        echo '{ "SINGULARITY_LABELS": "undefined" }'
    fi
fi

if [ -n "${SINGULARITY_INSPECT_DEFFILE:-}" ]; then
    if [ -f "$SINGULARITY_MOUNTPOINT/.singularity.d/Singularity" ]; then
        message 1 "## BOOTSTRAP DEFINITION FILE:\n"
        cat "$SINGULARITY_MOUNTPOINT/.singularity.d/Singularity"
        echo
    else
        message ERROR "This container does not include the bootstrap definition\n"
    fi

fi

if [ -n "${SINGULARITY_INSPECT_RUNSCRIPT:-}" ]; then
    if [ -f "$SINGULARITY_MOUNTPOINT/.singularity.d/runscript" ]; then
        message 1 "## RUNSCRIPT:\n"
        cat "$SINGULARITY_MOUNTPOINT/.singularity.d/runscript"
        echo
    else
        message ERROR "This container does not have any runscript defined\n"
    fi

fi

if [ -n "${SINGULARITY_INSPECT_TEST:-}" ]; then
    if [ -f "$SINGULARITY_MOUNTPOINT/.singularity.d/test" ]; then
        message 1 "## TEST:\n"
        cat "$SINGULARITY_MOUNTPOINT/.singularity.d/test"
        echo
    else
        message ERROR "This container does not have any tests defined\n"
    fi

fi

if [ -n "${SINGULARITY_INSPECT_ENVIRONMENT:-}" ]; then
    if [ -f "$SINGULARITY_MOUNTPOINT/.singularity.d/environment" ]; then
        message 1 "## ENVIRONMENT:\n"
        cat "$SINGULARITY_MOUNTPOINT/.singularity.d/environment"
        echo
    else
        message ERROR "This container does not have any custom environment defined\n"
    fi

fi

