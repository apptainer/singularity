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


########## BEGIN BOOTSTRAP SCRIPT ##########

umask 0002

if ! DEBOOTSTRAP_PATH=`singularity_which debootstrap`; then
    message ERROR "debootstrap is not in PATH... Perhaps 'apt-get install' it?\n"
    exit 1
fi

if [ -n "${ARCH:-}" ]; then
    ARCH=`echo ${ARCH:-} | sed -e 's/\s//g'`
else
    ARCH=`uname -m`

    if [ "$ARCH" == "x86_64" ]; then
        ARCH=amd64
    elif [ "$ARCH" == "ppc64le" ]; then
        ARCH=ppc64el
    elif [ "$ARCH" == "aarch64" ]; then
        ARCH=arm64
    elif [ "$ARCH" == "armv6l" ]; then
        ARCH=armhf
    elif [ "$ARCH" == "armv7l" ]; then
        ARCH=armhf
    fi
fi


if [ -z "${MIRRORURL:-}" ]; then
    message ERROR "No 'MirrorURL' defined in bootstrap definition\n"
    ABORT 1
fi

if [ -z "${OSVERSION:-}" ]; then
    message ERROR "No 'OSVersion' defined in bootstrap definition\n"
    ABORT 1
fi

REQUIRES=`echo "${INCLUDE:-}" | sed -e 's/\s/,/g'`

# The excludes save 25M or so with jessie.  (Excluding udev avoids
# systemd, for instance.)  There are a few more we could exclude
# to save a few MB.  I see 182M cleaned with this, v. 241M with
# the default debootstrap.
if ! eval "$DEBOOTSTRAP_PATH --variant=minbase --exclude=openssl,udev,debconf-i18n,e2fsprogs --include=apt,$REQUIRES --arch=$ARCH '$OSVERSION' '$SINGULARITY_ROOTFS' '$MIRRORURL'"; then
    ABORT 255
fi
