#!/bin/sh -
# Copyright (c) Sylabs Inc. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be found
# in the LICENSE file.
set -e

package_name=singularity
tree_version=`(git describe --match 'v[0-9]*' --always 2>/dev/null || cat VERSION 2>/dev/null || echo "") | sed -e "s/^v//;s/-/_/g;s/_/-/;s/_/./g"`
package_version=`(git describe --abbrev=0 --match 'v[0-9]*' --always 2>/dev/null || cat VERSION 2>/dev/null || echo "") | sed -e "s/^v//;s/-/_/g;s/_/-/;s/_/./g"`

echo " DIST setup VERSION: $tree_version"
echo $tree_version > VERSION
git add VERSION
# spec file needs to be at the root of the project
cp dist/rpm/singularity.spec .
git add singularity.spec
echo " DIST create tarball: $package_name-$package_version.tar.gz"
git archive --format=tar --prefix=$package_name/ `git stash create` -o $package_name-$package_version.tar
git reset VERSION singularity.spec
gzip $package_name-$package_version.tar

