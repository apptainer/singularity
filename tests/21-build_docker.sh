#!/bin/bash
#
# Copyright (c) 2017, Michael W. Bauer. All rights reserved.
# Copyright (c) 2017, Gregory M. Kurtzer. All rights reserved.
#
# "Singularity" Copyright (c) 2016, The Regents of the University of California,
# through Lawrence Berkeley National Laboratory (subject to receipt of any
# required approvals from the U.S. Dept. of Energy).  All rights reserved.
#
# This software is licensed under a customized 3-clause BSD license.  Please
# consult LICENSE file distributed with the sources of this project regarding
# your rights to use or distribute this software.
#
# NOTICE.  This Software was developed under funding from the U.S. Department of
# Energy and the U.S. Government consequently retains certain rights. As such,
# the U.S. Government has been granted for itself and others acting on its
# behalf a paid-up, nonexclusive, irrevocable, worldwide license in the Software
# to reproduce, distribute copies to the public, prepare derivative works, and
# perform publicly and display publicly, and to permit other to do so.
#
#



. ./functions

test_init "Docker bootstrap tests"



CONTAINER="$SINGULARITY_TESTDIR/container.img"
DEFFILE="$SINGULARITY_TESTDIR/example.def"

# Make sure the examples/docker/Singularity is pointing to busybox:latest (nobody mess with the examples! LOL)
stest 0 grep busybox:latest ../examples/docker/Singularity

stest 0 cp ../examples/docker/Singularity "$DEFFILE"
stest 0 sudo singularity build "$CONTAINER" "$DEFFILE"

stest 0 sed -i -e 's@busybox:latest@ubuntu:latest@' "$DEFFILE"
stest 0 sudo singularity build -F "$CONTAINER" "$DEFFILE"
stest 0 singularity exec "$CONTAINER" true
stest 1 singularity exec "$CONTAINER" false

stest 0 sed -i -e 's@ubuntu:latest@centos:latest@' "$DEFFILE"
stest 0 sudo singularity build -F "$CONTAINER" "$DEFFILE"
stest 0 singularity exec "$CONTAINER" true
stest 1 singularity exec "$CONTAINER" false

stest 0 sed -i -e 's@centos:latest@dock0/arch:latest@' "$DEFFILE"
stest 0 sudo singularity build -F "$CONTAINER" "$DEFFILE"
stest 0 singularity exec "$CONTAINER" true
stest 1 singularity exec "$CONTAINER" false

stest 0 sudo singularity build -F "$CONTAINER" docker://busybox
stest 0 singularity exec "$CONTAINER" true
stest 1 singularity exec "$CONTAINER" false

stest 1 sudo singularity build -F "$CONTAINER" docker://something_that_doesnt_exist_ever
stest 1 singularity exec "$CONTAINER" true
stest 1 singularity exec "$CONTAINER" false

if singularity_which docker >/dev/null 2>&1; then
# make sure local test does not exist, ignore errors
sudo docker kill registry >/dev/null 2>&1
sudo docker rm registry >/dev/null 2>&1

# start local docker registry
stest 0 sudo docker run -d -p 5000:5000 --restart=always --name registry registry:2
# pull busybox from docker and push to local registry
stest 0 sudo docker pull busybox
stest 0 sudo docker tag busybox localhost:5000/my-busybox
stest 0 sudo docker push localhost:5000/my-busybox

# alright, now we have a local registry to test with

# test with custom registry and custom namespace (including empty namespace)
# from squashfs to squashfs (via def file)
cat >"$DEFFILE" <<EOF
Bootstrap: docker
From: localhost:5000/my-busybox
EOF

stest 0 sudo SINGULARITY_NOHTTPS=true singularity build -F "$CONTAINER" "$DEFFILE"
stest 0 singularity exec "$CONTAINER" true
stest 1 singularity exec "$CONTAINER" false

cat >"$DEFFILE" <<EOF
Bootstrap: docker
From: my-busybox
Registry: localhost:5000
EOF

# this will fail, as it will try to use the default namespace
stest 1 sudo SINGULARITY_NOHTTPS=true singularity build -F "$CONTAINER" "$DEFFILE"

cat >"$DEFFILE" <<EOF
Bootstrap: docker
From: my-busybox
Registry: localhost:5000
Namespace:
EOF

# remove registry container, not needed now
stest 0 sudo SINGULARITY_NOHTTPS=true singularity build -F "$CONTAINER" "$DEFFILE"
stest 0 singularity exec "$CONTAINER" true
stest 1 singularity exec "$CONTAINER" false
# note: technically this doesn't clean everything, the volume the registry
# created, containing out pushed my-busybox, is still there

# destroy registry container once done
stest 0 sudo docker kill registry
stest 0 sudo docker rm registry

fi

test_cleanup
