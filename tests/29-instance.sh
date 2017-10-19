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

if [ ! -d "/proc/self/ns" ]; then
    echo "Instance is not supported on your host, skipping tests"
    exit 0
fi

test_init "Instance command group tests"

CONTAINER="$SINGULARITY_TESTDIR/container"

stest 0 sudo singularity build "$CONTAINER" "../examples/busybox/Singularity"
stest 0 singularity -x -d instance.start "$CONTAINER" service1
stest 0 sleep 5
stest 0 singularity -x exec instance://service1 true
stest 1 singularity -x exec instance://service1 false

stest 1 singularity instance.start "$CONTAINER" service1
stest 0 singularity instance.start "$CONTAINER" service2
stest 0 singularity instance.start "$CONTAINER" service3
stest 0 singularity instance.start "$CONTAINER" t1
stest 0 singularity instance.start "$CONTAINER" t2
stest 0 singularity instance.start "$CONTAINER" t22
stest 0 singularity instance.start "$CONTAINER" t3
stest 0 singularity instance.start "$CONTAINER" t4
stest 0 singularity instance.list service1
stest 0 singularity instance.stop service1
stest 1 singularity instance.list service1
stest 0 singularity instance.stop service\*
stest 1 singularity instance.list service\*
stest 0 singularity instance.list
stest 0 singularity instance.list t\*
stest 0 singularity instance.stop t1 t2\* t3
stest 0 singularity instance.list t\*
stest 0 singularity instance.stop --all
stest 1 singularity instance.list t\*


stest 0 sudo rm -rf "$CONTAINER"
test_cleanup
