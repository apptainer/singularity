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

# when calling this script, please use the following syntax to clean env
# env -i ./record-env.sh $SINGULARITY_ROOTFS
# because env should be clean upon entry, no sanity checking or functions
SINGULARITY_ROOTFS=${1:-}
for file in $(ls ${SINGULARITY_ROOTFS}/.singularity.d/env/*.sh); do
    . ${file} > /dev/null 2>&1
done
tmpfile=$(mktemp)
printenv | sort >$tmpfile
echo $tmpfile
