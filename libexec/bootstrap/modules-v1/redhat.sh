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

# dnf should probably be preferred if it's present
if INSTALL_CMD=`singularity_which dnf`; then
    message 1 "Found DNF at: $INSTALL_CMD\n"
elif INSTALL_CMD=`singularity_which yum`; then
    message 1 "Found YUM at: $INSTALL_CMD\n"
    INSTALL_CMD="$INSTALL_CMD --tolerant"
else
    message ERROR "Neither yum nor dnf in PATH!\n"
    exit 1
fi

REPO_COUNT=0
YUM_CONF="/etc/bootstrap-yum.conf"
export YUM_CONF

# Create the main portion of yum config
mkdir -p "$SINGULARITY_ROOTFS"

YUM_CONF_DIRNAME=`dirname $YUM_CONF`
mkdir -m 0755 -p "$SINGULARITY_ROOTFS/$YUM_CONF_DIRNAME"

> "$SINGULARITY_ROOTFS/$YUM_CONF"
echo "[main]" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
echo 'cachedir=/var/cache/yum/$basearch/$releasever' >> "$SINGULARITY_ROOTFS/$YUM_CONF"
echo "keepcache=0" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
echo "debuglevel=2" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
echo "logfile=/var/log/yum.log" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
echo "exactarch=1" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
echo "obsoletes=1" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
echo "gpgcheck=0" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
echo "plugins=1" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
echo "reposdir=0" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
echo "" >> "$SINGULARITY_ROOTFS/$YUM_CONF"


OSVersion() {
    return 0
}


MirrorURL() {
    if [ -n "${2:-}" ]; then
        REPO_NAME="${2:-}"
    else
        REPO_NAME="repo-${REPO_COUNT}"
        REPO_COUNT=`expr $REPO_COUNT + 1`
    fi
    echo "[$REPO_NAME]" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
    echo 'name=Linux $releasever - $basearch' >> "$SINGULARITY_ROOTFS/$YUM_CONF"
    echo "baseurl=${1:-}" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
    echo "enabled=1" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
    echo "gpgcheck=0" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
    echo "" >> "$SINGULARITY_ROOTFS/$YUM_CONF"

    return 0
}

Bootstrap() {
    # Avoid plugins which might cuase trouble, e.g. etckeeper, Red Hat
    # subscription-manager.  Install the release file, as the name of
    # the release package varies between RHEL, CentOS, etc.
    if ! eval "$INSTALL_CMD --noplugins -c $SINGULARITY_ROOTFS/$YUM_CONF --installroot $SINGULARITY_ROOTFS -y install /etc/redhat-release coreutils $@"; then
        exit 1
    fi

    __mountproc
    __mountsys
    __mountdev

    return 0
}

InstallPkgs() {
    IPCONF=''
    if [ -f $SINGULARITY_ROOTFS/$YUM_CONF ]; then
        IPCONF="-c $SINGULARITY_ROOTFS/$YUM_CONF"
    fi

    if ! eval "$INSTALL_CMD $IPCONF --noplugins --nogpgcheck --installroot $SINGULARITY_ROOTFS -y install $@"; then
        exit 1
    fi

    return 0
}

Cleanup() {
    if ! eval "$INSTALL_CMD --noplugins --installroot $SINGULARITY_ROOTFS clean all"; then
        exit 1
    fi

    # Remove RPM locks
    rm -f "$SINGULARITY_ROOTFS/var/lib/rpm/__*"

    return 0
}
