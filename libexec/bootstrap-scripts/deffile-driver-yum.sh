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

install -d -m 0755 "$SINGULARITY_ROOTFS/dev"

cp -a /dev/null         "$SINGULARITY_ROOTFS/dev/null"      2>/dev/null || > "$SINGULARITY_ROOTFS/dev/null"
cp -a /dev/zero         "$SINGULARITY_ROOTFS/dev/zero"      2>/dev/null || > "$SINGULARITY_ROOTFS/dev/zero"
cp -a /dev/random       "$SINGULARITY_ROOTFS/dev/random"    2>/dev/null || > "$SINGULARITY_ROOTFS/dev/random"
cp -a /dev/urandom      "$SINGULARITY_ROOTFS/dev/urandom"   2>/dev/null || > "$SINGULARITY_ROOTFS/dev/urandom"


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
    if [ -f "/etc/redhat-release" ]; then
        OSVERSION=`rpm -qf --qf '%{VERSION}' /etc/redhat-release`
    else
        OSVERSION=7
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
fi
echo 'cachedir=/var/cache/yum-bootstrap' >> "$SINGULARITY_ROOTFS/$YUM_CONF"
echo "keepcache=0" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
echo "debuglevel=2" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
echo "logfile=/var/log/yum.log" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
echo "syslog_device=/dev/null" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
echo "exactarch=1" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
echo "obsoletes=1" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
if [ -n "${GPG:-}" ]; then
	echo "gpgcheck=1" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
else
	echo "gpgcheck=0" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
fi
echo "plugins=1" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
echo "reposdir=0" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
echo "deltarpm=0" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
echo "" >> "$SINGULARITY_ROOTFS/$YUM_CONF"

echo "[base]" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
echo 'name=Linux $releasever - $basearch' >> "$SINGULARITY_ROOTFS/$YUM_CONF"
if [ -n "${MIRROR:-}" ]; then
echo "baseurl=$MIRROR" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
fi
if [ -n "${MIRROR_META:-}" ]; then
    echo "metalink=$MIRROR_META" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
fi
echo "enabled=1" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
if [ -n "${GPG:-}" ]; then
	echo "gpgcheck=1" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
else
	echo "gpgcheck=0" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
fi


if [ -n "${MIRROR_UPDATES:-}" ] || [ -n "${MIRROR_UPDATES_META:-}" ]; then
echo "[updates]" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
echo 'name=Linux $releasever - $basearch updates' >> "$SINGULARITY_ROOTFS/$YUM_CONF"
if [ -n "${MIRROR_UPDATES:-}" ]; then
echo "baseurl=${MIRROR_UPDATES}" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
fi
if [ -n "${MIRROR_UPDATES_META:-}" ]; then
    echo "metalink=$MIRROR_UPDATES_META" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
fi
echo "enabled=1" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
if [ -n "${GPG:-}" ]; then
	echo "gpgcheck=1" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
else
	echo "gpgcheck=0" >> "$SINGULARITY_ROOTFS/$YUM_CONF"
fi
fi

echo "" >> "$SINGULARITY_ROOTFS/$YUM_CONF"


# If GPG is specified, then we need to import a key from somewhere.
if [ -n "${GPG:-}" ]; then
  message 1 "We have a GPG key!  Preparing RPM database.\n"
  if ! eval rpm --root $SINGULARITY_ROOTFS --initdb; then
    message ERROR "Failed to create rpmdb!"
    ABORT 255
  fi

  # RPM will import from the web, if curl is installed, so check for it.
  if [ ${GPG:0:8} = 'https://' ]; then
    if CURL_CMD=`singularity_which curl`; then
      message 1 "Found curl at: $CURL_CMD\n"
    else
      message ERROR "curl not in PATH!\n"
      ABORT 1
    fi
  fi

  # Before importing, check for (and fail on) HTTP URLs.
  # Then let RPM handle everything for us!
  if [ ${GPG:0:7} = 'http://' ]; then
    message ERROR "It is unsafe to fetch a GPG key with an HTTP URL.\n"
    ABORT 255
  else
    if ! eval $RPM_CMD --root $SINGULARITY_ROOTFS --import $GPG; then
      message ERROR "Failed to import downloaded GPG key.\n"
      ABORT 255
    fi
    message 1 "GPG key import complete!\n"
  fi
else
  message 1 "Skipping GPG key import.\n"
fi

# Do the install!
if ! eval "$INSTALL_CMD --noplugins -c $SINGULARITY_ROOTFS/$YUM_CONF --installroot $SINGULARITY_ROOTFS --releasever=${OSVERSION} -y install /etc/redhat-release coreutils ${INCLUDE:-}"; then
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
