#!/bin/bash
# 
# Copyright (c) 2017-2018, SyLabs, Inc. All rights reserved.
# 
# See the COPYRIGHT.md file at the top-level directory of this distribution and at
# https://github.com/sylabs/singularity/blob/master/COPYRIGHT.md.
# 
# This file is part of the Singularity Linux container project. It is subject to the license
# terms in the LICENSE.md file found in the top-level directory of this distribution and
# at https://github.com/sylabs/singularity/blob/master/LICENSE.md. No part
# of Singularity, including this file, may be copied, modified, propagated, or distributed
# except according to the terms contained in the LICENSE.md file.
#
#

. ./functions

test_init "Environment tests"

# No Dockerfile custom path, No SINGULARITYENV_* variables 
stest 0 singularity exec docker://alpine:3.8 env | grep -q \
    PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

# Dockerfile custom path, No SINGULARITYENV_* variables 
stest 0 singularity exec docker://godlovedc/lolcow env | grep -q \
    PATH=/usr/games:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

# No Dockerfile custom path, Set SINGULARITYENV_PREPEND_PATH
export SINGULARITYENV_PREPEND_PATH=/foo
stest 0 singularity exec docker://alpine:3.8 env | grep -q \
    PATH=/foo:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

# Dockerfile custom path, Set SINGULARITYENV_PREPEND_PATH
stest 0 singularity exec docker://godlovedc/lolcow env | grep -q \
    PATH=/foo:/usr/games:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

# No Dockerfile custom path, Set SINGULARITYENV_APPEND_PATH
export SINGULARITYENV_APPEND_PATH=/bar
stest 0 singularity exec docker://alpine:3.8 env | grep -q \
    PATH=/foo:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/bar

# Dockerfile custom path, Set SINGULARITYENV_APPEND_PATH
stest 0 singularity exec docker://godlovedc/lolcow env | grep -q \
    PATH=/foo:/usr/games:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/bar

# No Dockerfile custom path, Set SINGULARITYENV_PATH
export SINGULARITYENV_PATH=/usr/bin:/bin
stest 0 singularity exec docker://alpine:3.8 env | grep -q \
    PATH=/usr/bin:/bin

# Dockerfile custom path, Set SINGULARITYENV_PATH
stest 0 singularity exec docker://godlovedc/lolcow env | grep -q \
    PATH=/usr/bin:/bin

test_cleanup
