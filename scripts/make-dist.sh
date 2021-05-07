#!/bin/sh -
# Copyright (c) 2018-2021, Sylabs Inc. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be found
# in the LICENSE file.
set -e

package_name=singularity-ce

if [ ! -f $package_name.spec ]; then
    echo "Run this from the top of the source tree after mconfig" >&2
    exit 1
fi

version=$1

if test -z "${version}" ; then
    cat >&2 <-EOT
	This program requires a version number as argument.
	
	        $0 {version}
	EOT
    exit 1
fi

echo " DIST setup VERSION: ${version}"
echo "${version}" > VERSION
rmfiles="VERSION"
tarball="${package_name}-${version}.tar.gz"
echo " DIST create tarball: $tarball"
rm -f $tarball
pathtop="${package_name}-${version}"
ln -sf .. builddir/$pathtop
rmfiles="$rmfiles builddir/$pathtop"
trap "rm -f $rmfiles" 0

# modules should have been vendored using the correct version of the Go
# tool, so we expect to find a vendor directory. Bail out if there isn't
# one.
if test ! -d vendor ; then
    echo 'E: vendor directory not found. Abort.'
    exit 1
fi

# XXX(mem): In order to accept filenames with colons in it (because of a
# version number like x.y.z:1.2.3), pass the --force-local flag to tar.
# This is understood by GNU tar. If other tar programs (also called
# "tar") don't, this will need to be fixed.
(echo VERSION; echo $package_name.spec; echo vendor; git ls-files) | \
    sed "s,^,$pathtop/," |
    tar --force-local -C builddir -T - -czf "$tarball"
