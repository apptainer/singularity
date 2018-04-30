#!/bin/sh -
set -e

topdir=$PWD
coredir=$topdir/core
buildtree=$coredir/buildtree

CONFIG_PKG="github.com/singularityware/singularity/pkg/configs"
CONFIG_LDFLAGS="-s -w"

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
# Go portion
#
CGO_CPPFLAGS="$CGO_CPPFLAGS -I$buildtree -I$coredir -I$coredir/lib"
CGO_LDFLAGS="$CGO_LDFLAGS -L$buildtree/lib -Wl,--unresolved-symbols=ignore-in-object-files"
export CGO_CPPFLAGS CGO_LDFLAGS

gen_go_constants() {
    config="`mktemp -u`.go"

    echo "package constants" > $config
    echo "// #include \"$buildtree/config.h\"" >> $config
    echo "import \"C\"" >> $config

    for i in $(cat $buildtree/config.h | cut -d " " -f 2); do
        echo "const $i = C.$i" >> $config
    done

    cd $topdir
    mkdir -p pkg/configs/constants
    go tool cgo -objdir /tmp -godefs $config > pkg/configs/constants/constants.go
    rm -f $config
    cd $coredir
}

#
# Singularity core C portion (libsycore.a)
#
if [ -d "$buildtree" -a -f "$buildtree/Makefile" ]; then
	make -j `nproc 2>/dev/null || echo 1` -C $buildtree
else
	cd $coredir
	./mconfig -b $buildtree
    gen_go_constants
    go build -ldflags="-s -w" -buildmode=c-archive -o $buildtree/librpc.a $coredir/runtime/go/rpc.go $coredir/runtime/go/smaster.go $coredir/runtime/go/scontainer.go
	make -j `nproc 2>/dev/null || echo 1` -C $buildtree
	cd $topdir
fi

go build -ldflags "${CONFIG_LDFLAGS}" --tags "containers_image_openpgp" -o $buildtree/singularity $topdir/cmd/cli/cli.go
go build -ldflags "${CONFIG_LDFLAGS}" -o $buildtree/sbuild $topdir/cmd/sbuild/sbuild.go
#go build -ldflags "${CONFIG_LDFLAGS}" -o $buildtree/scontainer $coredir/runtime/go/scontainer.go
#go build -ldflags "${CONFIG_LDFLAGS}" -o $buildtree/smaster $coredir/runtime/go/smaster.go

sudo cp $buildtree/wrapper $buildtree/wrapper-suid
sudo chown root:root $buildtree/wrapper-suid && sudo chmod 4755 $buildtree/wrapper-suid

