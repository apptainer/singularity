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

test_init "Instance command group tests"

CONTAINER="$SINGULARITY_TESTDIR/container"

stest 0 sudo singularity build "$CONTAINER" "../examples/busybox/Singularity"
stest 0 singularity instance.start "$CONTAINER" service1
stest 0 sh -c "ps -ef | grep -q sinit"
stest 0 sh -c "singularity exec --join service1 ${CONTAINER} ps -ef | grep -q sinit"
export PID=`singularity exec --join service1 ${CONTAINER} ps -ef | grep sinit | awk '{print $1}'`
stest 0 sh -c "echo $PID | grep -q 1" 
stest 1 singularity instance.start "$CONTAINER" service1
stest 0 singularity instance.start "$CONTAINER" service2
stest 0 singularity instance.start "$CONTAINER" service3
stest 0 sh -c "ps -ef | grep sinit | wc -l | grep -q 4" # 3 instances + grep process
stest 0 singularity instance.list
stest 0 sh -c "singularity instance.list | grep service | wc -l | grep -q 3"
stest 0 singularity instance.stop "$CONTAINER" service1
stest 0 sh -c "ps -ef | grep sinit | wc -l | grep -q 4" # 2 instances + grep process + subshell
stest 0 singularity instance.stop-all
stest 0 sh -c "ps -ef | grep sinit | wc -l | grep -q 2" # just grep process and subshell
stest 0 sh -c "singularity instance.list | wc -l | grep -q 0"

stest 0 sudo rm -rf "$CONTAINER"
test_cleanup
