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

test_init "Checking environment"

stest 0 sudo true
stest 1 sudo false
stest 0 which singularity
stest 0 test -f "$SINGULARITY_sysconfdir/singularity/singularity.conf"

# Check whether singularity binary is on a volume mounted with 'nosuid'
# I guess one could just grep /etc/mtab, but...
mount | grep $(df -h $SINGULARITY_PATH | grep -v File | awk '{print $1}') \
    > $SINGULARITY_TESTDIR/singularity_fs_nosuid
stest 1 grep "nosuid" $SINGULARITY_TESTDIR/singularity_fs_nosuid

# Is the SINGULARITY_PATH going to be there when you sudo?
sudo grep "^Defaults.*secure_path=.*$SINGULARITY_PATH" /etc/sudoers
echo $? > $SINGULARITY_TESTDIR/singularity_secure_path
stest 0 grep 0 $SINGULARITY_TESTDIR/singularity_secure_path

# Yes, Virginia, we use curl in these tests
stest 0 which curl

test_cleanup
