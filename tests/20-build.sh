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

test_init "Bootstrap tests"



CONTAINER="$SINGULARITY_TESTDIR/container"
CONTAINERIMG="$SINGULARITY_TESTDIR/container.img"
CONTAINERDIR="$SINGULARITY_TESTDIR/container.dir"


which singularity

stest 0 sudo singularity build "$CONTAINER" "../examples/busybox/Singularity"
stest 0 singularity exec "$CONTAINER" true
stest 1 singularity exec "$CONTAINER" false

stest 0 singularity exec "$CONTAINER" test -f /.singularity.d/runscript
stest 0 singularity exec "$CONTAINER" test -f /.singularity.d/labels.json
stest 0 singularity exec "$CONTAINER" test -f /.singularity.d/env/01-base.sh
stest 0 singularity exec "$CONTAINER" test -f /.singularity.d/actions/shell
stest 0 singularity exec "$CONTAINER" test -f /.singularity.d/actions/exec
stest 0 singularity exec "$CONTAINER" test -f /.singularity.d/actions/run
stest 0 singularity exec "$CONTAINER" test -L /environment
stest 0 singularity exec "$CONTAINER" test -L /singularity


# This should fail as root does not own the parent directory
stest 1 sudo singularity build --sandbox "$CONTAINERDIR" "../examples/busybox/Singularity"
# Force fixes that
stest 0 sudo singularity -x build --force --sandbox "$CONTAINERDIR" "../examples/busybox/Singularity"
stest 0 singularity exec "$CONTAINERDIR" true
stest 1 singularity exec "$CONTAINERDIR" false

stest 0 singularity exec "$CONTAINERDIR" test -f /.singularity.d/runscript
stest 0 singularity exec "$CONTAINERDIR" test -f /.singularity.d/labels.json
stest 0 singularity exec "$CONTAINERDIR" test -f /.singularity.d/env/01-base.sh
stest 0 singularity exec "$CONTAINERDIR" test -f /.singularity.d/actions/shell
stest 0 singularity exec "$CONTAINERDIR" test -f /.singularity.d/actions/exec
stest 0 singularity exec "$CONTAINERDIR" test -f /.singularity.d/actions/run
stest 0 singularity exec "$CONTAINERDIR" test -L /environment
stest 0 singularity exec "$CONTAINERDIR" test -L /singularity


stest 0 sudo singularity build --writable "$CONTAINERIMG" "../examples/busybox/Singularity"
stest 0 singularity exec "$CONTAINERIMG" true
stest 1 singularity exec "$CONTAINERIMG" false

stest 0 singularity exec "$CONTAINERIMG" test -f /.singularity.d/runscript
stest 0 singularity exec "$CONTAINERIMG" test -f /.singularity.d/labels.json
stest 0 singularity exec "$CONTAINERIMG" test -f /.singularity.d/env/01-base.sh
stest 0 singularity exec "$CONTAINERIMG" test -f /.singularity.d/actions/shell
stest 0 singularity exec "$CONTAINERIMG" test -f /.singularity.d/actions/exec
stest 0 singularity exec "$CONTAINERIMG" test -f /.singularity.d/actions/run
stest 0 singularity exec "$CONTAINERIMG" test -L /environment
stest 0 singularity exec "$CONTAINERIMG" test -L /singularity


stest 0 singularity build -F "$CONTAINER" docker://busybox
stest 0 singularity exec "$CONTAINER" true
stest 1 singularity exec "$CONTAINER" false

stest 0 singularity exec "$CONTAINER" test -f /.singularity.d/runscript
stest 0 singularity exec "$CONTAINER" test -f /.singularity.d/labels.json
stest 0 singularity exec "$CONTAINER" test -f /.singularity.d/env/01-base.sh
stest 0 singularity exec "$CONTAINER" test -f /.singularity.d/actions/shell
stest 0 singularity exec "$CONTAINER" test -f /.singularity.d/actions/exec
stest 0 singularity exec "$CONTAINER" test -f /.singularity.d/actions/run
stest 0 singularity exec "$CONTAINER" test -L /environment
stest 0 singularity exec "$CONTAINER" test -L /singularity


test_cleanup
