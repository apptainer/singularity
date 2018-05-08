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

test_init "Import/Export tests"



CONTAINER="$SINGULARITY_TESTDIR/container.img"

stest 0 touch "$SINGULARITY_TESTDIR/hello_world"
stest 0 singularity image.create -s 32 "$CONTAINER"
stest 0 sh -c "tar cf - -C $SINGULARITY_TESTDIR hello_world | sudo singularity image.import $CONTAINER"
stest 0 /bin/rm "$SINGULARITY_TESTDIR/hello_world"
stest 0 sh -c "singularity image.export $CONTAINER | tar xf - -C $SINGULARITY_TESTDIR"
stest 0 test -f "$SINGULARITY_TESTDIR/hello_world"

test_cleanup
