#!/bin/sh
#
#  Copyright (c) 2018, Sylabs, Inc. All rights reserved.
#
#  This software is licensed under a 3-clause BSD license.  Please
#  consult LICENSE file distributed with the sources of this project regarding
#  your rights to use or distribute this software.
#

export GOPATH=$PWD/go:$(go env GOPATH)
go build -o ../build/scontainer go/scontainer.go
go build -buildmode=c-shared -o ../build/librpc.so go/rpc.go
go build -o ../build/smaster go/smaster.go
go build -o cli tmpdev/cli.go

gcc c/wrapper.c c/util/message.c -o ../build/wrapper -Ic -I../build -L../build -ldl
sudo rm -f /tmp/wrapper-suid /tmp/wrapper /tmp/scontainer /tmp/smaster /tmp/librpc.so
cp ../build/wrapper /tmp/
cp ../build/wrapper /tmp/wrapper-suid
cp ../build/scontainer /tmp/
cp ../build/smaster /tmp/
cp ../build/librpc.so /tmp/
sudo chown root:root /tmp/wrapper-suid && sudo chmod 4755 /tmp/wrapper-suid

if [ ! -e "/tmp/testing.simg" ]; then
    singularity pull --name testing.simg shub://GodloveD/busybox
    mv testing.simg /tmp/
fi
if [ ! -e "/tmp/testing" ]; then
    mkdir -p /tmp/testing
    curl -s http://dl-cdn.alpinelinux.org/alpine/v3.7/releases/x86_64/alpine-minirootfs-3.7.0-x86_64.tar.gz | tar xzf - -C /tmp/testing 2>/dev/null
fi
