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

test_init "Checking escalation block"



CONTAINER="$SINGULARITY_TESTDIR/container.img"

stest 0 sudo singularity build --sandbox "$CONTAINER" docker://centos:7
stest 0 sudo singularity exec -w "$CONTAINER" chmod +s /bin/ping

stest 0 singularity exec "$CONTAINER" true
stest 1 singularity exec "$CONTAINER" false

# Checking no new privs with capabilities
stest 1 sudo singularity exec "$CONTAINER" ping -c 1 127.0.0.1
stest 1 singularity exec "$CONTAINER" ping -c 1 127.0.0.1

stest 1 sudo singularity exec --keep-privs "$CONTAINER" su -s /bin/sh - bin -c "ping -c 1 127.0.0.1"
stest 0 sudo singularity exec --keep-privs --allow-setuid "$CONTAINER" su -s /bin/sh - bin -c "ping -c 1 127.0.0.1"

stest 1 sudo singularity exec "$CONTAINER" mount -B /etc /mnt
stest 1 sudo singularity exec --no-privs "$CONTAINER" mount -B /etc /mnt
stest 0 sudo singularity exec --add-caps sys_admin "$CONTAINER" mount -B /etc /mnt

stest 0 sudo singularity exec --no-privs --add-caps sys_admin "$CONTAINER" mount -B /etc /mnt
stest 1 sudo singularity exec --keep-privs --drop-caps sys_admin "$CONTAINER" mount -B /etc /mnt

stest 1 sudo singularity exec "$CONTAINER" dd if=/dev/mem of=/dev/null bs=1 count=1
stest 0 sudo singularity exec --keep-privs "$CONTAINER" dd if=/dev/mem of=/dev/null bs=1 count=1

stest 0 sudo rm -rf "$CONTAINER"

test_cleanup

