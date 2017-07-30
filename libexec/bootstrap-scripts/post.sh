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
# This file also contains content that is covered under the LBNL/DOE/UC modified
# 3-clause BSD license and is subject to the license terms in the LICENSE-LBNL.md
# file found in the top-level directory of this distribution and at
# https://github.com/singularityware/singularity/blob/master/LICENSE-LBNL.md.


## Basic sanity
if [ -z "$SINGULARITY_libexecdir" ]; then
    echo "Could not identify the Singularity libexecdir."
    exit 1
fi

## Load functions
if [ -f "$SINGULARITY_libexecdir/singularity/functions" ]; then
    . $SINGULARITY_libexecdir/singularity/functions
else
    echo "Error loading functions: $SINGULARITY_libexecdir/singularity/functions"
    exit 1
fi

if [ -z "${SINGULARITY_ROOTFS:-}" ]; then
    message ERROR "Singularity root file system not defined\n"
    exit 1
fi


message 1 "Finalizing Singularity container\n"

umask 0002

test -L "$SINGULARITY_ROOTFS/etc/mtab"  && rm -f "$SINGULARITY_ROOTFS/etc/mtab"

cat > "$SINGULARITY_ROOTFS/etc/mtab" << EOF
singularity / rootfs rw 0 0
EOF


# Populate the labels.
# NOTE: We have to be careful to quote stuff that we know isn't quoted.
SINGULARITY_LABELFILE=$(printf "%q" "$SINGULARITY_ROOTFS/.singularity.d/labels.json")
SINGULARITY_ADD_SCRIPT=$(printf "%q" "$SINGULARITY_libexecdir/singularity/python/helpers/json/add.py")


eval $SINGULARITY_ADD_SCRIPT -f --key SINGULARITY_DEFFILE --value $(printf "%q" "$SINGULARITY_BUILDDEF") --file $SINGULARITY_LABELFILE

eval $SINGULARITY_ADD_SCRIPT -f --key SINGULARITY_BOOTSTRAP_VERSION --value $(printf "%q" "$SINGULARITY_version") --file $SINGULARITY_LABELFILE

env | egrep "^SINGULARITY_DEFFILE_" | while read i; do
    KEY=`echo $i | cut -f1 -d =`
    VAL=`echo $i | cut -f2- -d =`
    eval $SINGULARITY_ADD_SCRIPT -f --key $(printf "%q" "$KEY") --value $(printf "%q" "$VAL") --file $SINGULARITY_LABELFILE

done

