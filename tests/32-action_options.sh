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
TESTDIR="$SINGULARITY_TESTDIR/home_test"

# Creating a new container
stest 0 sudo singularity build "$CONTAINER" "../examples/busybox/Singularity"
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
stest 1 singularity exec --pwd /non-existent-dir "$CONTAINER" true
stest 0 sh -c "singularity exec --pwd /etc '$CONTAINER' pwd | egrep '^/etc'"

# Testing --home
stest 0 mkdir -p "$TESTDIR"
stest 0 touch "$TESTDIR/testfile"
stest 0 singularity exec --home "$TESTDIR" "$CONTAINER" test -f "$TESTDIR/testfile"
stest 0 singularity exec --home "$TESTDIR:/home" "$CONTAINER" test -f "/home/testfile"
if [ -n "${SINGULARITY_OVERLAY_FS:-}" ]; then
    stest 0 singularity exec --contain --home "$TESTDIR:/blah" "$CONTAINER" test -f "/blah/testfile"
fi
stest 0 sh -c "echo 'cd; test -f testfile' | singularity exec --home '$TESTDIR' '$CONTAINER' /bin/sh"
stest 1 singularity exec --home "/tmp" "$CONTAINER" true
stest 1 singularity exec --home "/tmp:/home" "$CONTAINER" true


test_cleanup
