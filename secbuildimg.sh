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


prefix="/usr/local"
exec_prefix="${prefix}"
libexecdir="${exec_prefix}/libexec"
sysconfdir="${prefix}/etc"
localstatedir="${prefix}/var"
bindir="${exec_prefix}/bin"

SINGULARITY_libexecdir="$libexecdir"
SINGULARITY_PATH="$bindir"

SECBUILD_IMAGE="$SINGULARITY_libexecdir/singularity/bootstrap-scripts/secbuild.img"

if [ $(id -ru) != 0 ]; then
    echo
    echo "You must be root to install"
    echo
    exit 1
fi

if [ -e "$SECBUILD_IMAGE" ]; then
    rm -f $SECBUILD_IMAGE
fi

BUILDTMP=$(mktemp -d)
SECBUILD_SINGULARITY="$BUILDTMP/singularity"
SECBUILD_DEFFILE="$BUILDTMP/secbuild.def"

cat > "$SECBUILD_DEFFILE" << DEFFILE
Bootstrap: docker
From: debian:jessie-slim

%setup
    cp -a . $SECBUILD_SINGULARITY

%post 
    export LC_LANG=C
    apt-get update
    apt-get install -y git gcc python make
    apt-get install -y squashfs-tools
    apt-get install -y debootstrap yum zypper
    apt-get clean
    apt-get autoclean
    cd $SECBUILD_SINGULARITY
    ./configure --disable-suid
    make clean
    make -j4
    make install
DEFFILE

$SINGULARITY_PATH/singularity build $SECBUILD_IMAGE $SECBUILD_DEFFILE

rm -rf $BUILDTMP

echo
echo "Secbuild image completed"
