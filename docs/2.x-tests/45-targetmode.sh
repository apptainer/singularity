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

test_init "Checking target UID/GID mode"



CONTAINER="$SINGULARITY_TESTDIR/container.img"

stest 0 sudo singularity build "$CONTAINER" "../examples/busybox/Singularity"
stest 0 singularity exec "$CONTAINER" true
stest 1 singularity exec "$CONTAINER" false

stest 0 sh -c "sudo SINGULARITY_TARGET_GID=`id -g` SINGULARITY_TARGET_UID=`id -u` singularity exec $CONTAINER whoami | grep `id -un`"
stest 1 sh -c "SINGULARITY_TARGET_GID=99 SINGULARITY_TARGET_UID=99 singularity exec $CONTAINER whoami | grep 99"
stest 1 sh -c "SINGULARITY_TARGET_GID=99 SINGULARITY_TARGET_UID=99 singularity exec $CONTAINER true"


test_cleanup

