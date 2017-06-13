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

test_init "Checking expand"


CONTAINER="$SINGULARITY_TESTDIR/container.img"
TMPMNT="$SINGULARITY_TESTDIR/mnt"

exit_cleanup() {
    sudo umount "$TMPMNT"
}


stest 0 mkdir "$TMPMNT"
stest 0 singularity create -s 5 "$CONTAINER"
stest 0 sudo singularity mount -s "$CONTAINER" "$TMPMNT"
stest 1 sudo dd if=/dev/zero of="$TMPMNT/file1" bs=1 count=5
stest 0 sudo umount "$TMPMNT"

stest 0 sudo singularity mount -s -w "$CONTAINER" "$TMPMNT"
stest 0 sudo dd if=/dev/zero of="$TMPMNT/file1" bs=1M count=3
stest 1 sudo dd if=/dev/zero of="$TMPMNT/file2" bs=1M count=3
stest 0 sudo rm -f "$TMPMNT/file2"
stest 0 sudo umount "$TMPMNT"

stest 0 singularity expand -s 5 "$CONTAINER"
stest 0 sudo singularity mount -s -w "$CONTAINER" "$TMPMNT"
stest 0 sudo dd if=/dev/zero of="$TMPMNT/file2" bs=1M count=3
stest 0 sudo umount "$TMPMNT"

test_cleanup

