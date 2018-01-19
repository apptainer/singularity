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

test_init "Basic container action tests"



CONTAINER="$SINGULARITY_TESTDIR/container.img"

# Creating a new container
stest 0 sudo singularity build "$CONTAINER" "../examples/busybox/Singularity"

# Testing shell command
stest 0 singularity shell "$CONTAINER" -c "true"
stest 0 sh -c "echo true | singularity shell '$CONTAINER'"
stest 1 singularity shell "$CONTAINER" -c "false"
stest 1 sh -c "echo false | singularity shell '$CONTAINER'"

# Testing exec command
stest 0 singularity exec "$CONTAINER" true
stest 0 singularity exec "$CONTAINER" /bin/true
stest 1 singularity exec "$CONTAINER" false
stest 1 singularity exec "$CONTAINER" /bin/false
stest 1 singularity exec "$CONTAINER" /blahh
stest 1 singularity exec "$CONTAINER" blahh
stest 0 sh -c "echo hi | singularity exec $CONTAINER grep hi"
stest 1 sh -c "echo bye | singularity exec $CONTAINER grep hi"


# Testing run command
stest 0 singularity run "$CONTAINER" true
stest 1 singularity run "$CONTAINER" false


# Testing run command properly hands arguments
stest 0 sh -c "singularity run '$CONTAINER' foo | grep foo"


# Testing singularity properly handles STDIN
stest 0 sh -c "echo true | singularity shell '$CONTAINER'"
stest 1 sh -c "echo false | singularity shell '$CONTAINER'"
stest 0 sh -c "echo true | singularity exec '$CONTAINER' /bin/sh"
stest 1 sh -c "echo false | singularity exec '$CONTAINER' /bin/sh"


# Checking permissions
stest 0 sh -c "singularity exec $CONTAINER id -u | grep `id -u`"
stest 0 sh -c "sudo singularity exec $CONTAINER id -u | grep 0"


test_cleanup

