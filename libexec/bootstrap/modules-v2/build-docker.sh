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

# Ensure the user has provided a docker image name with "From"
if [ -z "$SINGULARITY_DOCKER_IMAGE" ]; then
    echo "Please specify the Docker image name with From: in the definition file."
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
    message ERROR "Singularity build definition file not defined\n"
    exit 1
fi


########## BEGIN BOOTSTRAP SCRIPT ##########


: ' ADMIN IMAGE STUFFS -------------------------------------------
 Here we create the admin account, and define hosts
'

mkdir -p -m 0755 "$SINGULARITY_ROOTFS/bin"
mkdir -p -m 0755 "$SINGULARITY_ROOTFS/etc"

echo "root:!:0:0:root:/root:/bin/sh" > "$SINGULARITY_ROOTFS/etc/passwd"
echo " root:x:0:" > "$SINGULARITY_ROOTFS/etc/group"
echo "127.0.0.1   localhost localhost.localdomain localhost4 localhost4.localdomain4" > "$SINGULARITY_ROOTFS/etc/hosts"

# TODO: if made into official module, export to pythonpath here
#TODO: at install, python dependencies need to be installed, and check for python
python $SINGULARITY_libexecdir/singularity/helpers/bootstrap.py --docker $SINGULARITY_DOCKER_IMAGE --rootfs $SINGULARITY_ROOTFS

# If we got here, exit...
exit 0
