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


# dnf should probably be preferred if it's present, at some point we will make
# a dnf specific bootstrap module.
if INSTALL_CMD=`singularity_which dnf`; then
    message 1 "Found DNF at: $INSTALL_CMD\n"
elif INSTALL_CMD=`singularity_which yum`; then
    message 1 "Found YUM at: $INSTALL_CMD\n"
    INSTALL_CMD="$INSTALL_CMD --tolerant"
else
    message ERROR "Neither yum nor dnf in PATH!\n"
    ABORT 1
fi

OSVERSION=`singularity_key_get "OSVersion" "$SINGULARITY_BUILDDEF"`
if [ -z "${OSVERSION:-}" ]; then
    if [ -f "/etc/redhat-release" ]; then
        OSVERSION=`rpm -qf --qf '%{VERSION}' /etc/redhat-release`
    else
        OSVERSION=7
    fi
fi

MIRROR=`singularity_key_get "MirrorURL" "$SINGULARITY_BUILDDEF" | sed -r "s/%\{?OSVERSION\}?/$OSVERSION/gi"`
if [ -z "${MIRROR:-}" ]; then
    message ERROR "No 'MirrorURL' defined in bootstrap definition\n"
    ABORT 1
fi
MIRROR_UPDATES=`singularity_key_get "UpdateURL" "$SINGULARITY_BUILDDEF" | sed -r "s/%\{?OSVERSION\}?/$OSVERSION/gi"`
if [ ! -z "${MIRROR_UPDATES:-}" ]; then
    message 1 "'UpdateURL' defined in bootstrap definition\n"
fi

INSTALLPKGS=`singularity_keys_get "Include" "$SINGULARITY_BUILDDEF"`

REPO_COUNT=0
YUM_CONF="/etc/bootstrap-yum.conf"
export YUM_CONF

# Create the main portion of yum config
mkdir -p "$SINGULARITY_ROOTFS"

YUM_CONF_DIRNAME=`dirname $YUM_CONF`
mkdir -m 0755 -p "$SINGULARITY_ROOTFS/$YUM_CONF_DIRNAME"

> "$SINGULARITY_ROOTFS/$YUM_CONF"
echo "[main]" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
if [ -n "${http_proxy:-}" ]; then
    echo "proxy=$http_proxy" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
elif [ -n "${HTTP_PROXY:-}" ]; then
    echo "proxy=$HTTP_PROXY" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
fi
echo 'cachedir=/var/cache/yum-bootstrap' >> "$SINGULARITY_ROOTFS/$YUM_CONF"
echo "keepcache=0" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
echo "debuglevel=2" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
echo "logfile=/var/log/yum.log" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
echo "syslog_device=/dev/null" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
echo "exactarch=1" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
echo "obsoletes=1" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
echo "gpgcheck=0" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
echo "plugins=1" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
echo "reposdir=0" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
echo "" >> "$SINGULARITY_ROOTFS/$YUM_CONF"

echo "[base]" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
echo 'name=Linux $releasever - $basearch' >> "$SINGULARITY_ROOTFS/$YUM_CONF"
echo "baseurl=$MIRROR" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
echo "enabled=1" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
echo "gpgcheck=0" >> "$SINGULARITY_ROOTFS/$YUM_CONF"

if [ ! -z "${MIRROR_UPDATES:-}" ]; then
echo "[updates]" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
echo 'name=Linux $releasever - $basearch updates' >> "$SINGULARITY_ROOTFS/$YUM_CONF"
echo "baseurl=${MIRROR_UPDATES}" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
echo "enabled=1" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
echo "gpgcheck=0" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
fi

echo "" >> "$SINGULARITY_ROOTFS/$YUM_CONF"

if ! eval "$INSTALL_CMD --noplugins -c $SINGULARITY_ROOTFS/$YUM_CONF --installroot $SINGULARITY_ROOTFS -y install /etc/redhat-release coreutils $INSTALLPKGS"; then
    message ERROR "Bootstrap failed... exiting\n"
    ABORT 255
fi

if [ -f "/etc/yum.conf" ]; then
    if [ -n "${http_proxy:-}" ]; then
        sed -i -e "s/\[main\]/\[main\]\nproxy=$http_proxy/" /etc/yum.conf
    elif [ -n "${HTTP_PROXY:-}" ]; then
        sed -i -e "s/\[main\]/\[main\]\nproxy=$HTTP_PROXY/" /etc/yum.conf
    fi
fi

if ! eval "rm -rf $SINGULARITY_ROOTFS/var/cache/yum-bootstrap"; then
    message WARNING "Failed cleaning Bootstrap packages\n"
fi

# If we got here, exit...
exit 0
