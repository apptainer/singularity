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
SINGULARITY_CONTAINER=`singularity_key_get "From" "$SINGULARITY_BUILDDEF"`
if [ -z "${SINGULARITY_CONTAINER:-}" ]; then
    message ERROR "Bootstrap type 'docker' given, but no 'From' defined!\n"
    ABORT 1
else
    export SINGULARITY_CONTAINER
    message 1 "From: $SINGULARITY_CONTAINER\n"
fi

### Obtain the IncludeCmd from the spec (also needed for docker bootstrap)
SINGULARITY_DOCKER_INCLUDE_CMD=`singularity_key_get "IncludeCmd" "$SINGULARITY_BUILDDEF"`
if [ -n "${SINGULARITY_DOCKER_INCLUDE_CMD:-}" ]; then
    message 1 "IncludeCmd: $SINGULARITY_DOCKER_INCLUDE_CMD\n"

    # A command of "yes" means that we will include the docker CMD as runscript
    if [ "$SINGULARITY_DOCKER_INCLUDE_CMD" == "yes" ]; then
        export SINGULARITY_DOCKER_INCLUDE_CMD
    fi
fi


### Does the registry require authentication?
SINGULARITY_DOCKER_USERNAME=`singularity_key_get "Username" "$SINGULARITY_BUILDDEF"`
SINGULARITY_DOCKER_PASSWORD=`singularity_key_get "Password" "$SINGULARITY_BUILDDEF"`
if [ -n "${SINGULARITY_DOCKER_USERNAME:-}" ] && [ -n "${SINGULARITY_DOCKER_PASSWORD:-}" ]; then
    message 1 "Username: $SINGULARITY_DOCKER_USERNAME\n"
    message 1 "Password: [hidden]\n"
    export SINGULARITY_DOCKER_USERNAME SINGULARITY_DOCKER_PASSWORD
fi


# Ensure the user has provided a docker image name with "From"
if [ -z "$SINGULARITY_CONTAINER" ]; then
    echo "Please specify the Docker image name with From: in the definition file."
    exit 1
fi

eval $SINGULARITY_libexecdir/singularity/python/docker/import.py 

# If we got here, exit...
exit 0
