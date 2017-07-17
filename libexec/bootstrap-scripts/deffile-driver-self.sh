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

if [ -z "${SINGULARITY_BUILDDEF:-}" ]; then
    exit
fi


########## BEGIN BOOTSTRAP SCRIPT ##########

umask 0002


if ! GUNZIP_PATH=`singularity_which gunzip`; then
    message ERROR "gunzip is not in PATH... Perhaps 'apt-get install' it?\n"
    exit 1
fi


# By default, we clone from root unless specified otherwise

if [ -z "${FROM:-}" ]; then
    FROM='/'
fi

message 1 "Cloning from $FROM\m"
message 1 "Preparing contents to bootstrap image by self clone with base $FROM\n"
SINGULARITY_DUMP=`mktemp /tmp/.singularity-layers.XXXXXXXX.tgz`
export SINGULARITY_DUMP

# The user can specify custom exclusions

if [ -z "${EXCLUDE:-}" ]; then
    EXCLUDE=''
else
    message 1 "Custom exclusions: $EXCLUDE\n"
fi
CUSTOM_EXCLUSIONS=$(echo "$EXCLUDE" | sed 's/[^ ]* */--exclude &/g')

# Extract the host into a container
tar --one-file-system -czvSf $SINGULARITY_DUMP --exclude $SINGULARITY_DUMP --exclude $HOME --exclude $SINGULARITY_libexecdir --exclude ${TMPDIR-/tmp} --exclude $SINGULARITY_libexecdir/singularity $CUSTOM_EXCLUSIONS --exclude /usr/src $FROM

eval_abort "$SINGULARITY_libexecdir/singularity/bootstrap-scripts/pre.sh"
eval_abort "$SINGULARITY_libexecdir/singularity/bootstrap-scripts/environment.sh"

message 1 "Extracting self into new image\n"

cd $SINGULARITY_ROOTFS && gunzip -dc $SINGULARITY_DUMP

rm -f "$SINGULARITY_DUMP"

eval_abort "$SINGULARITY_libexecdir/singularity/bootstrap-scripts/post.sh"
