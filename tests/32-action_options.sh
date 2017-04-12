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

test_init "Testing action options"



CONTAINER="$SINGULARITY_TESTDIR/container.img"

# Creating a new container
stest 0 singularity create -s 568 "$CONTAINER"
stest 0 sudo singularity bootstrap "$CONTAINER" "../examples/busybox/Singularity"
stest 0 singularity exec "$CONTAINER" true
stest 1 singularity exec "$CONTAINER" false

# Checking if Singularity properly handles custom shells
stest 0 singularity shell -s /bin/true "$CONTAINER"
stest 1 singularity shell -s /bin/false "$CONTAINER"

# Testing --workdir
stest 0 touch "$SINGULARITY_TESTDIR/testfile"
stest 0 singularity exec --workdir "$SINGULARITY_TESTDIR" "$CONTAINER" test -f "$SINGULARITY_TESTDIR/testfile"
stest 1 singularity exec --workdir "$SINGULARITY_TESTDIR" --contain "$CONTAINER" test -f "$SINGULARITY_TESTDIR/testfile"

# Testing --pwd
stest 0 singularity exec --pwd /etc "$CONTAINER" true
stest 1 singularity exec --pwd /non-existant-dir "$CONTAINER" true
stest 0 sh -c "singularity exec --pwd /etc '$CONTAINER' pwd | egrep '^/etc'"



test_cleanup
