#!/bin/bash
# 
# Copyright (c) 2017, FlyElephant Team, MIT License.
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

install -d -m 0755 "$SINGULARITY_ROOTFS/dev"

cp -a /dev/null         "$SINGULARITY_ROOTFS/dev/null"      2>/dev/null || > "$SINGULARITY_ROOTFS/dev/null"
cp -a /dev/zero         "$SINGULARITY_ROOTFS/dev/zero"      2>/dev/null || > "$SINGULARITY_ROOTFS/dev/zero"
cp -a /dev/random       "$SINGULARITY_ROOTFS/dev/random"    2>/dev/null || > "$SINGULARITY_ROOTFS/dev/random"
cp -a /dev/urandom      "$SINGULARITY_ROOTFS/dev/urandom"   2>/dev/null || > "$SINGULARITY_ROOTFS/dev/urandom"


# dnf should probably be preferred if it's present, at some point we will make
# a dnf specific bootstrap module.
if INSTALL_CMD=`singularity_which zypper`; then
    message 1 "Found Zypper at: $INSTALL_CMD\n"
    INSTALL_CMD="$INSTALL_CMD -n"
else
    message ERROR "Neither zypper in PATH!\n"
    ABORT 1
fi

OSVERSION=`singularity_key_get "OSVersion" "$SINGULARITY_BUILDDEF"`
if [ -z "${OSVERSION:-}" ]; then
    OSVERSION=42.2
fi

MIRROR=`singularity_key_get "MirrorURL" "$SINGULARITY_BUILDDEF" | sed -r "s/%\{?OSVERSION\}?/$OSVERSION/gi"`
if [ -z "${MIRROR:-}" ]; then
    message ERROR "No 'MirrorURL' defined in bootstrap definition\n"
    ABORT 1
fi

# Create the main portion of yum config
mkdir -p "$SINGULARITY_ROOTFS"
$INSTALL_CMD --root $SINGULARITY_ROOTFS ar $MIRROR repo-oss
$INSTALL_CMD --root $SINGULARITY_ROOTFS --gpg-auto-import-keys refresh
$INSTALL_CMD --root $SINGULARITY_ROOTFS install patterns-openSUSE-base zypper

# If we got here, exit...
exit 0
