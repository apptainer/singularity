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


# the order of these tests is important because one container often
# builds from another

. ./functions

test_init "Build tests"


CONTAINER="$SINGULARITY_TESTDIR/container"
CONTAINER2="$SINGULARITY_TESTDIR/container2"


alias container_check="stest 0 singularity exec \"$CONTAINER\" true ; 
stest 1 singularity exec \"$CONTAINER\" false ; 
stest 0 singularity exec \"$CONTAINER\" test -f /.singularity.d/runscript ; 
stest 0 singularity exec \"$CONTAINER\" test -f /.singularity.d/labels.json ; 
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
# This should fail as root does not own the parent directory
stest 1 sudo singularity build --sandbox "$CONTAINER" "../examples/busybox/Singularity"
# Force fixes that
stest 0 sudo singularity -x build --force --sandbox "$CONTAINER" "../examples/busybox/Singularity"
container_check

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

# # from shub to squashfs (via def file)
# sudo rm "$CONTAINER"
# stest 0 sudo singularity build "$CONTAINER" "../examples/shub/Singularity"
# container_check

# from squashfs to squashfs (via def file)
cat >"${SINGULARITY_TESTDIR}/Singularity" <<EOF
Bootstrap: localimage
From: $CONTAINER2
EOF
sudo mv "$CONTAINER" "$CONTAINER2"
stest 0 sudo singularity build "$CONTAINER" "${SINGULARITY_TESTDIR}/Singularity"
container_check

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

# from def file to existing image 
sudo rm "$CONTAINER"
stest 0 singularity image.create "$CONTAINER"
stest 0 singularity build --exists "$CONTAINER" "..examples/busybox/Singularity"
container_check

# from tar to squashfs
singularity image.export "$CONTAINER" >"$CONTAINER2".tar
sudo rm "$CONTAINER"
stest 0 sudo singularity build "$CONTAINER" "$CONTAINER2".tar
container_check

# from tar.gx to squashfs
singularity image.export "$CONTAINER" | gzip -9 >"$CONTAINER2".tar.gz
sudo rm "$CONTAINER"
stest 0 sudo singularity build "$CONTAINER" "$CONTAINER2".tar.gz
container_check


stest 0 sudo rm -rf "${CONTAINER}"
stest 0 sudo rm -rf "${CONTAINER2}"
stest 0 sudo rm -rf "${CONTAINER2}".tar
stest 0 sudo rm -rf "${CONTAINER2}".tar.gz
stest 0 sudo rm -rf "${SINGULARITY_TESTDIR}/Singularity"

test_cleanup
