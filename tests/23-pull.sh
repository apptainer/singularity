#!/bin/bash
# 
# Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
# 
# See the COPYRIGHT.md file at the top-level directory of this distribution and at
# https://github.com/singularityware/singularity/blob/master/COPYRIGHT.md.
# 
# This file is part of the Singularity Linux container project. It is subject to the license
# terms in the LICENSE.md file found in the top-level directory of this distribution and
# at https://github.com/singularityware/singularity/blob/master/LICENSE.md. No part
# of Singularity, including this file, may be copied, modified, propagated, or distributed
# except according to the terms contained in the LICENSE.md file.
#
#


. ./functions

test_init "Pull tests"

cd "$SINGULARITY_TESTDIR"

stest 0 sudo singularity pull --size 10 docker://busybox
CONTAINER=busybox.simg
stest 0 singularity exec "$CONTAINER" true
stest 1 singularity exec "$CONTAINER" false
stest 0 singularity exec "$CONTAINER" test -f /.singularity.d/runscript
stest 0 singularity exec "$CONTAINER" test -f /.singularity.d/env/01-base.sh
stest 0 singularity exec "$CONTAINER" test -f /.singularity.d/actions/shell
stest 0 singularity exec "$CONTAINER" test -f /.singularity.d/actions/exec
stest 0 singularity exec "$CONTAINER" test -f /.singularity.d/actions/run
stest 0 singularity exec "$CONTAINER" test -L /environment
stest 0 singularity exec "$CONTAINER" test -L /singularity

# should fail b/c we already pulled busybox
stest 1 sudo singularity pull --size 10 docker://busybox

# force should fix
stest 0 sudo singularity pull --force --size 10 docker://busybox

stest 0 sudo rm -rf "${CONTAINER}"

stest 1 singularity pull docker://this_should_not/exist

test_cleanup
