#!/bin/sh -
set -e

topdir=$PWD
coredir=$topdir/core
buildtree=$coredir/buildtree

#
# Singularity core C portion (libsycore.a)
#
if [ -d "$buildtree" -a -f "$buildtree/Makefile" ]; then
	make -j `nproc 2>/dev/null || echo 1` -C $buildtree
else
	cd $coredir
	./mconfig -b $buildtree
	make -j `nproc 2>/dev/null || echo 1` -C $buildtree
	cd $topdir
fi

#
# Go portion
#
CGO_CPPFLAGS="$CGO_CPPFLAGS -I$buildtree -I$coredir -I$coredir/lib"
CGO_LDFLAGS="$CGO_LDFLAGS -L$buildtree/lib"
export CGO_CPPFLAGS CGO_LDFLAGS

go build -o $buildtree/singularity cmd/cli/cli.go
