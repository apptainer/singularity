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
if [ -f "$SINGULARITY_libexecdir/singularity/bootstrap-scripts/functions" ]; then
    . "$SINGULARITY_libexecdir/singularity/bootstrap-scripts/functions"
else
    echo "Error loading functions: $SINGULARITY_libexecdir/singularity/bootstrap-scripts/functions"
    exit 1
fi

if [ -z "${SINGULARITY_ROOTFS:-}" ]; then
    message ERROR "Singularity root file system not defined\n"
    exit 1
fi

if [ -z "${SINGULARITY_BUILDDEF:-}" ]; then
    message ERROR "Singularity bootstrap definition file not defined!\n"
    exit 1
fi

if [ ! -f "${SINGULARITY_BUILDDEF:-}" ]; then
    message ERROR "Singularity bootstrap definition file not found!\n"
    exit 1
fi

umask 0002

test -d "$SINGULARITY_ROOTFS/proc" || install -d -m 755 "$SINGULARITY_ROOTFS/proc"
test -d "$SINGULARITY_ROOTFS/sys" || install -d -m 755 "$SINGULARITY_ROOTFS/sys"
test -d "$SINGULARITY_ROOTFS/tmp" || install -d -m 755 "$SINGULARITY_ROOTFS/tmp"
test -d "$SINGULARITY_ROOTFS/dev" || install -d -m 755 "$SINGULARITY_ROOTFS/dev"

mount -r --no-mtab -t proc proc "$SINGULARITY_ROOTFS/proc"
mount -r --no-mtab -t sysfs sysfs "$SINGULARITY_ROOTFS/sys"
mount --no-mtab --rbind "/tmp" "$SINGULARITY_ROOTFS/tmp"
mount --no-mtab --rbind "/dev" "$SINGULARITY_ROOTFS/dev"

cp /etc/hosts           "$SINGULARITY_ROOTFS/etc/hosts"
cp /etc/resolv.conf     "$SINGULARITY_ROOTFS/etc/resolv.conf"

eval_abort "SINGULARITY_STAGE2=1 ${SINGULARITY_libexecdir}/singularity/bin/builddef"
