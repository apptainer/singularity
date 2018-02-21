#!/bin/sh
#
#  Copyright (c) 2018, Sylabs, Inc. All rights reserved.
#
#  This software is licensed under a 3-clause BSD license.  Please
#  consult LICENSE file distributed with the sources of this project regarding
#  your rights to use or distribute this software.
#

export GOPATH=$PWD/go:$(go env GOPATH)
go build -ldflags '-s -w -extldflags "-static"' -a -o ../build/scontainer go/scontainer.go
go build -buildmode=c-archive -ldflags '-s -w' -a -o ../build/librpc.a go/rpc.go
go build -ldflags '-s -w -extldflags "-static"' -a -o ../build/smaster go/smaster.go

gcc c/wrapper.c -o ../build/wrapper -Ic -I../build -L../build -lrpc -lpthread
rm -f ../build/wrapper-suid
cp ../build/wrapper ../build/wrapper-suid
sudo chown root:root ../build/wrapper-suid && sudo chmod 4755 ../build/wrapper-suid

if [ ! -e "/tmp/testing.simg" ]; then
    singularity pull --name testing.simg shub://GodloveD/busybox
    mv testing.simg /tmp/
fi
if [ ! -e "/tmp/testing" ]; then
    mkdir -p /tmp/testing
    curl -s http://dl-cdn.alpinelinux.org/alpine/v3.7/releases/x86_64/alpine-minirootfs-3.7.0-x86_64.tar.gz | tar xzf - -C /tmp/testing 2>/dev/null
fi
