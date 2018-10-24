#!/bin/sh -
# Copyright (c) Sylabs Inc. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be found
# in the LICENSE file.
set -e

# spec file needs to be at the root of the project
cp dist/rpm/singularity.spec .

package_name=singularity
package_version_short="`sed -n 's/^Version: //p' singularity.spec`"
tree_version="$package_version_short-`sed -n 's/^Release: \([^%]*\).*/\1/p' singularity.spec`"

echo " DIST setup VERSION: $tree_version"
echo $tree_version > VERSION
tarball="$package_name-$package_version_short.tar.gz"
echo " DIST create tarball: $tarball"
rm -f $tarball
(echo VERSION; echo singularity.spec; git ls-files) | \
        tar -T - --xform "s,^,$package_name/," -czf $tarball
rm -f VERSION singularity.spec
