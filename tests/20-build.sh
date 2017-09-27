#!/bin/bash
# 
# Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
# 
# See the COPYRIGHT.md file at the top-level directory of this distribution and at
# https://github.com/singularityware/singularity/blob/master/COPYRIGHT.md.
# 
# This file is part of the Singularity Linux container project. It is subject to the license
# terms in the LICENSE.md file found in the top-level directory of this distribution and
# at https://github.com/singularityware/singularity/blob/master/LICENSE.md. No part
# of Singularity, including this file, may be copied, modified, propagated, or distributed
# except according to the terms contained in the LICENSE.md file.
#
#


# the order of these tests is important because one container often
# builds from another

. ./functions

test_init "Build tests"


CONTAINER="$SINGULARITY_TESTDIR/container"
CONTAINER2="$SINGULARITY_TESTDIR/container2"


alias container_check="stest 0 singularity exec \"$CONTAINER\" true ; 
stest 1 singularity exec \"$CONTAINER\" false ; 
stest 0 singularity exec \"$CONTAINER\" test -f /.singularity.d/runscript ; 
stest 0 singularity exec \"$CONTAINER\" test -f /.singularity.d/env/01-base.sh ; 
stest 0 singularity exec \"$CONTAINER\" test -f /.singularity.d/actions/shell  ; 
stest 0 singularity exec \"$CONTAINER\" test -f /.singularity.d/actions/exec ; 
stest 0 singularity exec \"$CONTAINER\" test -f /.singularity.d/actions/run ; 
stest 0 singularity exec \"$CONTAINER\" test -L /environment ; 
stest 0 singularity exec \"$CONTAINER\" test -L /singularity"


# from definition file to squashfs
stest 0 sudo singularity build "$CONTAINER" "../examples/busybox/Singularity"
container_check

# from definition file to sandbox
sudo rm "$CONTAINER"
stest 0 sudo singularity build --sandbox "$CONTAINER" "../examples/busybox/Singularity"
container_check

# from ridicolus to squashfs
stest 1 sudo singularity build "$CONTAINER" "/some/dumb/path"

# from sandbox to squashfs
sudo mv "$CONTAINER" "$CONTAINER2"
stest 0 sudo singularity build "$CONTAINER" "$CONTAINER2"
container_check

# from definition file to image 
rm -rf "$CONTAINER"
stest 0 sudo singularity build --writable "$CONTAINER" "../examples/busybox/Singularity"
container_check

# from image to squasfs
sudo mv "$CONTAINER" "$CONTAINER2"
stest 0 sudo singularity build "$CONTAINER" "$CONTAINER2"
container_check

# from docker to squashfs
sudo rm "$CONTAINER"
stest 0 singularity build "$CONTAINER" "docker://busybox"
container_check

# from sqaushfs to squashfs 
sudo mv "$CONTAINER" "$CONTAINER2"
stest 0 sudo singularity build "$CONTAINER" "$CONTAINER2"
container_check

# from shub to squashfs 
sudo rm "$CONTAINER"
stest 0 singularity build "$CONTAINER" "shub://GodloveD/busybox"
container_check

# from docker to squashfs (via def file)
sudo rm "$CONTAINER"
stest 0 sudo singularity build "$CONTAINER" "../examples/docker/Singularity"
container_check

# from shub to squashfs (via def file)
sudo rm "$CONTAINER"
stest 0 sudo singularity build "$CONTAINER" "../examples/shub/Singularity"
container_check

# from squashfs to squashfs (via def file)
cat >"${SINGULARITY_TESTDIR}/Singularity" <<EOF
Bootstrap: localimage
From: $CONTAINER2
EOF
sudo mv "$CONTAINER" "$CONTAINER2"
stest 0 sudo singularity build "$CONTAINER" "${SINGULARITY_TESTDIR}/Singularity"
container_check

# with labels
cat >>"${SINGULARITY_TESTDIR}/Singularity" <<EOF
%labels
    FOO bar
EOF
sudo mv "$CONTAINER" "$CONTAINER2"
stest 0 sudo singularity build "$CONTAINER" "${SINGULARITY_TESTDIR}/Singularity"
container_check
stest 0 singularity exec "$CONTAINER" test -f /.singularity.d/labels.json

# from localimage to squashfs (via def file)
sudo rm -rf "$CONTAINER" "$CONTAINER2"
stest 0 sudo singularity build --writable "$CONTAINER2" "../examples/busybox/Singularity"
stest 0 sudo singularity build "$CONTAINER" "${SINGULARITY_TESTDIR}/Singularity"
container_check

# from sandbox to squashfs (via def file)
sudo rm -rf "$CONTAINER" "$CONTAINER2"
stest 0 sudo singularity -x build --force --sandbox "$CONTAINER2" "../examples/busybox/Singularity"
stest 0 sudo singularity build "$CONTAINER" "${SINGULARITY_TESTDIR}/Singularity"
container_check

# from tar to squashfs
stest 0 sudo sh -c "singularity image.export '$CONTAINER' > '${CONTAINER2}.tar'"
stest 0 sudo rm "$CONTAINER"
stest 0 sudo singularity build "$CONTAINER" "${CONTAINER2}.tar"
container_check

# from tar.gx to squashfs
stest 0 sh -c "singularity image.export '$CONTAINER' | gzip -9 > '${CONTAINER2}.tar.gz'"
sudo rm "$CONTAINER"
stest 0 sudo singularity build "$CONTAINER" "${CONTAINER2}.tar.gz"
container_check


stest 0 sudo rm -rf "${CONTAINER}"
stest 0 sudo rm -rf "${CONTAINER2}"
stest 0 sudo rm -rf "${CONTAINER2}".tar
stest 0 sudo rm -rf "${CONTAINER2}".tar.gz
stest 0 sudo rm -rf "${SINGULARITY_TESTDIR}/Singularity"

test_cleanup
