#!/bin/sh -
# Copyright (c) Sylabs Inc. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be found
# in the LICENSE file.
set -e

package_name=singularity
package_version_short="`sed -n 's/^Version: //p' singularity.spec`"
tree_version="$package_version_short-`sed -n 's/^Release: \([^%]*\).*/\1/p' singularity.spec`"

echo " DIST setup VERSION: $tree_version"
echo $tree_version > VERSION
rmfiles="VERSION"
tarball="$package_name-$package_version_short.tar.gz"
echo " DIST create tarball: $tarball"
rm -f $tarball
pathtop="$package_name"
thisdir="`basename $PWD`"
if [ "$thisdir" != "$pathtop" ]; then
    ln -fs $thisdir ../$pathtop
    rmfiles="$rmfiles ../$pathtop"
fi
trap "rm -f $rmfiles" 0
(echo VERSION; echo singularity.spec; git ls-files) | \
    sed "s,^,$pathtop/," | tar -C .. -T - -czf $tarball
