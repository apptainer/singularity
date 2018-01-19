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

if [ ! -d /usr/local/singularity-2.2.1-legacytests ]; then
    exit
fi

. ./functions

test_init "DISABLED: Checking new container with legacy Singularity"
#
#CONTAINER="$SINGULARITY_TESTDIR/container.img"
#CONTAINERDIR="$SINGULARITY_TESTDIR/container.dir"
#LEGACYDIR="/usr/local/singularity-2.2.1-legacytests/bin"
#
#stest 0 cp "../examples/busybox/Singularity" $SINGULARITY_TESTDIR
#stest 0 echo '%environment' >> "$SINGULARITY_TESTDIR/Singularity"
#stest 0 echo '    export FOO=bar' >> $SINGULARITY_TESTDIR"/Singularity"
#stest 0 sudo singularity build --writable "$CONTAINER" "$SINGULARITY_TESTDIR/Singularity"
#stest 0 singularity exec "$CONTAINER" true
#stest 1 singularity exec "$CONTAINER" false
#stest 0 $LEGACYDIR/singularity exec "$CONTAINER" true
#stest 1 $LEGACYDIR/singularity exec "$CONTAINER" false
#
#stest 0 $LEGACYDIR/singularity exec "$CONTAINER" test -f /.singularity.d/runscript
#stest 0 $LEGACYDIR/singularity exec "$CONTAINER" test -f /.singularity.d/labels.json
#stest 0 $LEGACYDIR/singularity exec "$CONTAINER" test -f /.singularity.d/env/01-base.sh
#stest 0 $LEGACYDIR/singularity exec "$CONTAINER" test -f /.singularity.d/actions/shell
#stest 0 $LEGACYDIR/singularity exec "$CONTAINER" test -f /.singularity.d/actions/exec
#stest 0 $LEGACYDIR/singularity exec "$CONTAINER" test -f /.singularity.d/actions/run
#stest 0 $LEGACYDIR/singularity exec "$CONTAINER" test -L /environment
#stest 0 $LEGACYDIR/singularity exec "$CONTAINER" test -L /singularity
#stest 0 $LEGACYDIR/singularity exec "$CONTAINER" env | grep -cw FOO=bar > /dev/null
#
#stest 0 mkdir "$CONTAINERDIR"
#stest 0 sudo singularity bootstrap "$CONTAINERDIR" "$SINGULARITY_TESTDIR/Singularity"
#stest 0 $LEGACYDIR/singularity exec "$CONTAINERDIR" true
#stest 1 $LEGACYDIR/singularity exec "$CONTAINERDIR" false
#stest 0 $LEGACYDIR/singularity exec "$CONTAINERDIR" env | grep -cw FOO=bar > /dev/null
#
#stest 0 sudo rm -rf "$CONTAINERDIR"

test_cleanup
