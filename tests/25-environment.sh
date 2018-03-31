#!/bin/bash
# 
# Copyright (c) 2017-2018, SyLabs, Inc. All rights reserved.
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

test_init "Environment tests"

# No Dockerfile custom path, No SINGULARITYENV_* variables 
stest 0 singularity exec docker://alpine env | grep -q \
    PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

# No Dockerfile custom path, Set SINGULARITYENV_PATH
stest 0 SINGULARITYENV_PATH=/usr/bin:/bin singularity exec docker://alpine env | grep -q \
    PATH=/usr/bin:/bin

# No Dockerfile custom path, Set SINGULARITYENV_APPEND_PATH
stest 0 SINGULARITYENV_APPEND_PATH=/opt singularity exec docker://alpine env | grep -q \
    PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/opt

# No Dockerfile custom path, Set SINGULARITYENV_PREPEND_PATH
stest 0 SINGULARITYENV_PREPEND_PATH=/opt singularity exec docker://alpine env | grep -q \
    PATH=/opt:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

# Dockerfile custom path, No SINGULARITYENV_* variables 
stest 0 singularity exec docker://godlovedc/lolcow env | grep -q \
    PATH=/usr/games:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

# Dockerfile custom path, Set SINGULARITYENV_PATH
stest 0 SINGULARITYENV_PATH=/usr/bin:/bin singularity exec docker://godlovedc/lolcow env | grep -q \
    PATH=/usr/bin:/bin

# Dockerfile custom path, Set SINGULARITYENV_APPEND_PATH
stest 0 SINGULARITYENV_APPEND_PATH=/testpath singularity exec docker://godlovedc/lolcow env | grep -q \
    PATH=/usr/games:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/testpath

# Dockerfile custom path, Set SINGULARITYENV_PREPEND_PATH
stest 0 SINGULARITYENV_PREPEND_PATH=/testpath singularity exec docker://godlovedc/lolcow env | grep -q \
    PATH=/testpath:/usr/games:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

test_cleanup
