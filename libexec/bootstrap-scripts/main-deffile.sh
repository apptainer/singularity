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

## For deffile bootstraps, we can implement a "minsize" guard.  This triggers a
## size check against the image we're trying to bootstrap into, so we don't
## waste time building into an image file that will be too small.
#this environment variable might not be defined, so we can make it null.
MINSIZE=${MINSIZE:-}
if [ -n "$MINSIZE" ]; then
  #report that it's been included in the def file.
  echo "Minimum image size has been set: $MINSIZE M"
  #get the size of the image, in megabytes.
  IMAGE_SIZE=$(du --apparent-size -m $SINGULARITY_IMAGE | cut -f 1)
  echo "Detected size of the image: $IMAGE_SIZE"
  #do the check.
  if [ "$IMAGE_SIZE" -lt "$MINSIZE" ]; then
    message ERROR "Size of $SINGULARITY_IMAGE insufficient to accomodate bootstrap minimum $MINSIZE M\n"
    exit 1
  fi
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

eval_abort "$SINGULARITY_libexecdir/singularity/bootstrap-scripts/deffile-sections.sh"
eval_abort "$SINGULARITY_libexecdir/singularity/bootstrap-scripts/post.sh"

# If checks specified, export variable
if [ "${SINGULARITY_CHECKS:-}" = "no" ]; then
    message 1 "Skipping checks\n"
else
    message 1 "Running checks\n"
    eval_abort "$SINGULARITY_libexecdir/singularity/bootstrap-scripts/checks.sh"
fi
