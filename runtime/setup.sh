#!/bin/sh
#
#  Copyright (c) 2018, Sylabs, Inc. All rights reserved.
#
#  This software is licensed under a 3-clause BSD license.  Please
#  consult LICENSE file distributed with the sources of this project regarding
#  your rights to use or distribute this software.
#

export GOPATH=$(go env GOPATH)
go build -ldflags="-s -w" -o ../build/scontainer go/scontainer.go
go build -ldflags="-s -w" -buildmode=c-shared -o ../build/librpc.so go/rpc.go
go build -ldflags="-s -w" -o ../build/smaster go/smaster.go
go build -ldflags="-s -w" -o cli tmpdev/cli.go

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
    singularity build --sandbox /tmp/testing docker://alpine
fi
