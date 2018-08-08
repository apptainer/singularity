#!/bin/bash
#
# Copyright (c) 2015-2016, Gregory M. Kurtzer. All rights reserved.
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

test_init "Action URI tests"



CONTAINER="$SINGULARITY_TESTDIR/container.img"

NO_XZ=false
if [ ! $(which xz) ]; then
    NO_XZ=true
    echo "Not testing with xz, not installed\n"
fi

# Testing Docker URI
stest 0 singularity exec docker://busybox true
stest 1 singularity exec docker://busybox false

# Creating a new container
stest 0 sudo singularity build "$CONTAINER" "../examples/busybox/Singularity"

# Creating tarball archives
stest 0 sh -c "singularity image.export "$CONTAINER" | gzip -c - > \"$CONTAINER.tar.gz\""
stest 0 sh -c "singularity image.export "$CONTAINER" | bzip2 -c - > \"$CONTAINER.tar.bz2\""

$NO_XZ || stest 0 sh -c "singularity image.export "$CONTAINER" | xz -c - > \"$CONTAINER.tar.xz\""

# Testing tarball archives
stest 0 singularity exec "$CONTAINER.tar.gz" true
stest 0 singularity exec "$CONTAINER.tar.bz2" true

$NO_XZ || stest 0 singularity exec "$CONTAINER.tar.xz" true

# Testing automatic algorithm detection
stest 0 mv "$CONTAINER.tar.gz" "$CONTAINER.tar.bz2"
stest 0 singularity exec "$CONTAINER.tar.bz2" true


test_cleanup

