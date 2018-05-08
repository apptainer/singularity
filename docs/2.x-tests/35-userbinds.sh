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

test_init "Testing user binds"



CONTAINER="$SINGULARITY_TESTDIR/container.img"

# Creating a new container
stest 0 sudo singularity build "$CONTAINER" "../examples/busybox/Singularity"

stest 0 touch /tmp/hello_world_test
stest 0 singularity exec -B /tmp:/opt "$CONTAINER" test -f /opt/hello_world_test

if [ -n "$SINGULARITY_OVERLAY_FS" ]; then
    stest 0 singularity exec -B /tmp:/nonexistent "$CONTAINER" test -f /nonexistent/hello_world_test
fi

stest 0 rm -f /tmp/hello_world_test

test_cleanup

