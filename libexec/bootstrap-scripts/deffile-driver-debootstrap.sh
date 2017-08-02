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
