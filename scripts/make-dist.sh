#!/bin/sh -
# Copyright (c) Sylabs Inc. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be found
# in the LICENSE file.
set -e

# spec file needs to be at the root of the project
cp dist/rpm/singularity.spec .
git add singularity.spec

package_name=singularity
package_version_short="`sed -n 's/^Version: //p' singularity.spec`"
tree_version="$package_version_short-`sed -n 's/^Release: \([^%]*\).*/\1/p' singularity.spec`"

# Remove Dist tarball if it exists
if [ -f $package_name-$package_version_short.tar.gz ]; then
    rm -f $package_name-$package_version_short.tar.gz
fi

echo " DIST setup VERSION: $tree_version"
echo $tree_version > VERSION
git add VERSION
echo " DIST create tarball: $package_name-$package_version_short.tar.gz"
git archive --format=tar --prefix=$package_name/ `git stash create` -o $package_name-$package_version_short.tar
git reset VERSION singularity.spec || true
gzip $package_name-$package_version_short.tar

