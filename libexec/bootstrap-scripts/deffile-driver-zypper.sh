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

install -d -m 0755 "$SINGULARITY_ROOTFS/dev"

cp -a /dev/null         "$SINGULARITY_ROOTFS/dev/null"      2>/dev/null || > "$SINGULARITY_ROOTFS/dev/null"
cp -a /dev/zero         "$SINGULARITY_ROOTFS/dev/zero"      2>/dev/null || > "$SINGULARITY_ROOTFS/dev/zero"
cp -a /dev/random       "$SINGULARITY_ROOTFS/dev/random"    2>/dev/null || > "$SINGULARITY_ROOTFS/dev/random"
cp -a /dev/urandom      "$SINGULARITY_ROOTFS/dev/urandom"   2>/dev/null || > "$SINGULARITY_ROOTFS/dev/urandom"


# dnf should probably be preferred if it's present, at some point we will make
# a dnf specific bootstrap module.
if INSTALL_CMD=`singularity_which zypper`; then
    message 1 "Found Zypper at: $INSTALL_CMD\n"
else
    message ERROR "Zypper not found in PATH!\n"
    ABORT 1
fi

# Check for RPM's dbpath not being /var/lib/rpm
RPM_CMD=`singularity_which rpm`
if [ -z "${RPM_CMD:-}" ]; then
    message ERROR "rpm not in PATH!\n"
    ABORT 1
fi
RPM_DBPATH=$(rpm --showrc | grep -E ":\s_dbpath\s" | cut -f2)
if [ "$RPM_DBPATH" != '%{_var}/lib/rpm' ]; then
    message ERROR "RPM database is using a weird path: %s\n" "$RPM_DBPATH"
    message WARNING "You are probably running this bootstrap on Debian or Ubuntu.\n"
    message WARNING "There is a way to work around this problem:\n"
    message WARNING "Create a file at path %s/.rpmmacros.\n" "$HOME"
    message WARNING "Place the following lines into the '.rpmmacros' file:\n"
    message WARNING "%s\n" '%_var /var'
    message WARNING "%s\n" '%_dbpath %{_var}/lib/rpm'
    message WARNING "After creating the file, re-run the bootstrap.\n"
    message WARNING "More info: https://github.com/singularityware/singularity/issues/241\n"
    ABORT 1
fi

if [ -z "${OSVERSION:-}" ]; then
    if [ -f "/etc/os-release" ]; then
        OSVERSION=`rpm -qf --qf '%{VERSION}' /etc/os-release`
    else
        OSVERSION=12.2
    fi
fi

MIRROR=`echo "${MIRRORURL:-}" | sed -r "s/%\{?OSVERSION\}?/$OSVERSION/gi"`
MIRROR_META=`echo "${METALINK:-}" | sed -r "s/%\{?OSVERSION\}?/$OSVERSION/gi"`
if [ -z "${MIRROR:-}" ] && [ -z "${MIRROR_META:-}" ]; then
    message ERROR "No 'MirrorURL' or 'MetaLink' defined in bootstrap definition\n"
    ABORT 1
 fi

MIRROR_UPDATES=`echo "${UPDATEURL:-}" | sed -r "s/%\{?OSVERSION\}?/$OSVERSION/gi"`
MIRROR_UPDATES_META=`echo "${UPDATEMETALINK:-}" | sed -r "s/%\{?OSVERSION\}?/$OSVERSION/gi"`
if [ -n "${MIRROR_UPDATES:-}" ] || [ -n "${MIRROR_UPDATES_META:-}" ]; then
    message 1 "'UpdateURL' or 'UpdateMetaLink' defined in bootstrap definition\n"
fi

ZYPP_CONF="/etc/zypp/zypp.conf"
export ZYPP_CONF

# Create the main portion of zypper config
mkdir -p "$SINGULARITY_ROOTFS"

ZYPP_CONF_DIRNAME=`dirname $ZYPP_CONF`
mkdir -m 0755 -p "$SINGULARITY_ROOTFS/$ZYPP_CONF_DIRNAME"

> "$SINGULARITY_ROOTFS/$ZYPP_CONF"
echo "[main]" >> "$SINGULARITY_ROOTFS/$ZYPP_CONF"
echo 'cachedir=/var/cache/zypp-bootstrap' >> "$SINGULARITY_ROOTFS/$ZYPP_CONF"
echo "" >> "$SINGULARITY_ROOTFS/$ZYPP_CONF"

# Import zypper repos
$INSTALL_CMD --root $SINGULARITY_ROOTFS ar $MIRROR repo-oss
$INSTALL_CMD --root $SINGULARITY_ROOTFS --gpg-auto-import-keys refresh

# Do the install!
if ! eval "$INSTALL_CMD -c $SINGULARITY_ROOTFS/$ZYPP_CONF --root $SINGULARITY_ROOTFS --releasever=${OSVERSION} -n install --auto-agree-with-licenses aaa_base ${INCLUDE:-}"; then
    message ERROR "Bootstrap failed... exiting\n"
    ABORT 255
fi

if ! eval "rm -rf $SINGULARITY_ROOTFS/var/cache/zypp-bootstrap"; then
    message WARNING "Failed cleaning Bootstrap packages\n"
fi

# If we got here, exit...
exit 0
