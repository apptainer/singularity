#!/bin/bash
#
# Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
# Copyright (c) 2016, Maciej Sieczka. All rights reserved
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


# Basic sanity.
if [ -z "$SINGULARITY_libexecdir" ]; then
    echo "Could not identify the Singularity libexecdir."
    exit 1
fi

# Load functions.
if [ -f "$SINGULARITY_libexecdir/singularity/functions" ]; then
    . "$SINGULARITY_libexecdir/singularity/functions"
else
    echo "Error loading functions: $SINGULARITY_libexecdir/singularity/functions."
    exit 1
fi

if [ -z "${SINGULARITY_ROOTFS:-}" ]; then
    message ERROR "Singularity root file system not defined.\n"
    ABORT 255
fi


########## BEGIN BOOTSTRAP SCRIPT ##########

if ! PACSTRAP=`singularity_which pacstrap`; then
    message ERROR "\`pacstrap' is not in PATH. You can install it with \`pacman -S arch-install-scripts'.\n"
    ABORT 1
fi

if ! WGET=`singularity_which wget`; then
    message ERROR "\`wget' is not in PATH. You can install it with \`pacman -S wget'.\n"
    ABORT 1
fi

ARCHITECTURE="`uname -m`"
if [ "$ARCHITECTURE" != 'x86_64' -a "$ARCHITECTURE" != 'i686' ]; then
    message ERROR "Architecture \`$ARCHITECTURE' is not supported."
    ABORT 1
fi

PACMAN_CONF_URL="https://git.archlinux.org/svntogit/packages.git/plain/trunk/pacman.conf.${ARCHITECTURE}?h=packages/pacman"

# `pacstrap' installs the whole "base" package group, unless told otherwise.
# $BASE_TO_SKIP are "base" packages that won't be normally needed on a
# container system. $BASE_TO_INST are "base" packages not present in
# $BASE_TO_SKIP. The list of packages included in "base" group may (it surely
# will, one day) change in future, so $BASE_TO_SKIP will need an update from
# time to time. Here I'm referring to `base' group contents as of 30.08.2016.
BASE_TO_SKIP='cryptsetup\|device-mapper\|dhcpcd\|iproute2\|jfsutils\|linux\|lvm2\|man-db\|man-pages\|mdadm\|nano\|netctl\|openresolv\|pciutils\|pcmciautils\|reiserfsprogs\|s-nail\|systemd-sysvcompat\|usbutils\|vi\|xfsprogs'
BASE_TO_INST=`pacman -Sgq base | grep -xv $BASE_TO_SKIP | tr '\n' ' '`

# TODO: Try choosing fastest mirror(s) with rankmirrors?
# https://wiki.archlinux.org/index.php/Mirrors#List_by_speed

PACMAN_CONF="/tmp/pacman.conf.$$"
# TODO: Use mktemp instead?
if ! eval "'$WGET' --no-verbose -O '$PACMAN_CONF' '$PACMAN_CONF_URL'"; then
    message ERROR "Failed to download \`$PACMAN_CONF_URL' to \`$PACMAN_CONF'.\n"
    ABORT 255
fi

# In addition to selected `base' packages `haveged' has to be installed. It's
# required to generate enough entropy for Pacman package signing setup without
# having to wait for ages until entropy accumulates. See
# https://wiki.archlinux.org/index.php/Install_from_Existing_Linux,
# https://wiki.archlinux.org/index.php/Pacman/Package_signing.
if ! eval "'$PACSTRAP' -C '$PACMAN_CONF' -c -d -G -M '$SINGULARITY_ROOTFS' haveged $BASE_TO_INST"; then
    rm -f "$PACMAN_CONF"
    message ERROR "\`$PACSTRAP' failed.\n"
    ABORT 255
fi

rm -f "$PACMAN_CONF"

# Pacman package signing setup.
if ! eval "arch-chroot '$SINGULARITY_ROOTFS' /bin/sh -c 'haveged -w 1024; pacman-key --init; pacman-key --populate archlinux'"; then
    message ERROR "Pacman package signing setup failed.\n"
    ABORT 255
fi

# Cleanup.
if ! eval "arch-chroot '$SINGULARITY_ROOTFS' pacman -Rs --noconfirm haveged"; then
    message ERROR "Bootstrap packages cleanup failed.\n"
    ABORT 255
fi
