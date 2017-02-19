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

if [ -z "${SINGULARITY_ROOTFS:-}" ]; then
    message ERROR "Singularity root file system not defined\n"
    exit 1
fi

if [ -z "${SINGULARITY_BUILDDEF:-}" ]; then
    exit
fi


########## BEGIN BOOTSTRAP SCRIPT ##########

### Obtain the From from the spec (needed for docker bootstrap)
export SINGULARITY_HUB_IMAGE=`singularity_key_get "From" "$SINGULARITY_BUILDDEF"`
if [ -z "${SINGULARITY_HUB_IMAGE:-}" ]; then
    message ERROR "Bootstrap type 'shub' given, but no 'From' defined!\n"
    ABORT 1
else
    message 1 "From: $SINGULARITY_HUB_IMAGE\n"
fi

# Ensure the user has provided a singularity hub id
if [ -z "$SINGULARITY_HUB_IMAGE" ]; then
    echo "Please specify the Singularity Hub Container ID with From: username/repo:tag in the definition file."
    exit 1
fi

if [ -n "${SINGULARITY_CACHEDIR:-}" ]; then
    SINGULARITY_CACHEDIR_LOCAL="$SINGULARITY_CACHEDIR"
else
    SINGULARITY_CACHEDIR_LOCAL="/tmp"
fi
if ! BASE_CONTAINER_DIR=`mktemp -d $SINGULARITY_CACHEDIR_LOCAL/singularity-container_dir.XXXXXXXX`; then
    message ERROR "Failed to create container_dir\n"
    ABORT 255
fi

export SINGULARITY_METADATA_DIR="$BASE_CONTAINER_DIR/$SINGULARITY_HUB_IMAGE"
if ! mkdir -p "$SINGULARITY_METADATA_DIR"; then
    message ERROR "Failed to create named container_dir\n"
    ABORT 255
fi

eval $SINGULARITY_libexecdir/singularity/python/shub/pull.py

# The python script saves names to files in CONTAINER_DIR, we then pass this image as targz to import
IMPORT_URI=`cat $SINGULARITY_METADATA_DIR/SINGULARITY_IMAGE`
rm $SINGULARITY_METADATA_DIR/SINGULARITY_IMAGE
rm $SINGULARITY_METADATA_DIR/SINGULARITY_RUNDIR
SINGULARITY_IMPORT_GET="cat $IMPORT_URI"
SINGULARITY_IMPORT_SPLAT="| ( cd '$SINGULARITY_ROOTFS' ; tar -xzf - )"
eval "$SINGULARITY_IMPORT_GET ${SINGULARITY_IMPORT_SPLAT:-}"

# If we got here, exit...
exit 0
