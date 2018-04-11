#!/bin/sh -
set -e

topdir=$PWD
coredir=$topdir/core
buildtree=$coredir/buildtree

CONFIG_PKG="github.com/singularityware/singularity/pkg/configs"
CONFIG_LDFLAGS="-X ${CONFIG_PKG}.BUILDTREE=${buildtree}"
CONFIG_LDFLAGS="${CONFIG_LDFLAGS} -X ${CONFIG_PKG}.LIBEXECDIR=/tmp/testing"

while true; do
    case ${1:-} in
	--clean)
	    sudo rm -rf $buildtree
	    shift
	;;
	*)
	    break;
	;;
    esac
done

#
# Singularity core C portion (libsycore.a)
#
if [ -d "$buildtree" -a -f "$buildtree/Makefile" ]; then
	make -j `nproc 2>/dev/null || echo 1` -C $buildtree
else
	cd $coredir
	./mconfig -b $buildtree
	go build -ldflags="-s -w" -buildmode=c-shared -o $buildtree/librpc.so $coredir/runtime/go/rpc.go
	make -j `nproc 2>/dev/null || echo 1` -C $buildtree
	cd $topdir
fi

#
# Go portion
#
CGO_CPPFLAGS="$CGO_CPPFLAGS -I$buildtree -I$coredir -I$coredir/lib"
CGO_LDFLAGS="$CGO_LDFLAGS -L$buildtree/lib"
export CGO_CPPFLAGS CGO_LDFLAGS

go build -ldflags "${CONFIG_LDFLAGS}" --tags "containers_image_openpgp" -o $buildtree/singularity $topdir/cmd/cli/cli.go
go build -ldflags "${CONFIG_LDFLAGS}" -o $buildtree/sbuild $topdir/cmd/sbuild/sbuild.go
go build -ldflags "${CONFIG_LDFLAGS}" -o $buildtree/scontainer $coredir/runtime/go/scontainer.go
go build -ldflags "${CONFIG_LDFLAGS}" -o $buildtree/smaster $coredir/runtime/go/smaster.go

sudo cp $buildtree/wrapper $buildtree/wrapper-suid
sudo chown root:root $buildtree/wrapper-suid && sudo chmod 4755 $buildtree/wrapper-suid

