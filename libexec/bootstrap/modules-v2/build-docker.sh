#!/bin/bash
# 
# Copyright (c) 2015-2016, Gregory M. Kurtzer. All rights reserved.
# 
# “Singularity” Copyright (c) 2016, The Regents of the University of California,
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
SINGULARITY_DOCKER_IMAGE=`singularity_key_get "From" "$SINGULARITY_BUILDDEF"`
if [ -z "${SINGULARITY_DOCKER_IMAGE:-}" ]; then
    message ERROR "Bootstrap type 'docker' given, but no 'From' defined!\n"
    ABORT 1
else
    message 1 "From: $SINGULARITY_DOCKER_IMAGE\n"
fi

### Obtain the IncludeCmd from the spec (also needed for docker bootstrap)
SINGULARITY_DOCKER_CMD=`singularity_key_get "IncludeCmd" "$SINGULARITY_BUILDDEF"`
if [ -n "${SINGULARITY_DOCKER_CMD:-}" ]; then
    message 1 "IncludeCmd: $SINGULARITY_DOCKER_CMD\n"

    # A command of "yes" means that we will include the docker CMD as runscript
    if [ "$SINGULARITY_DOCKER_CMD" == "yes" ]; then
        SINGULARITY_DOCKER_INCLUDE_CMD="--cmd"

    # Anything else, we will not include it
    else
        SINGULARITY_DOCKER_INCLUDE_CMD=""
    fi

# Default (not finding the IncludeCmd) is to not include
else
    SINGULARITY_DOCKER_INCLUDE_CMD=""
fi

# Ensure the user has provided a docker image name with "From"
if [ -z "$SINGULARITY_DOCKER_IMAGE" ]; then
    echo "Please specify the Docker image name with From: in the definition file."
    exit 1
fi

# Does the user want to include the docker CMD? Default, no.
if [ -z "$SINGULARITY_DOCKER_INCLUDE_CMD:-}" ]; then
    SINGULARITY_DOCKER_INCLUDE_CMD=""
fi

### Run it!

# TODO: if made into official module, export to pythonpath here
#TODO: at install, python dependencies need to be installed, and check for python
python $SINGULARITY_libexecdir/singularity/python/cli.py --docker $SINGULARITY_DOCKER_IMAGE --rootfs $SINGULARITY_ROOTFS $SINGULARITY_DOCKER_INCLUDE_CMD

# If we got here, exit...
exit 0
