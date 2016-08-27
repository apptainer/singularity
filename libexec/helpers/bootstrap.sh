#!/bin/bash
# 
# Copyright (c) 2015-2016, Gregory M. Kurtzer. All rights reserved.
# 
# “Singularity” Copyright (c) 2016, The Regents of the University of California,
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


## Basic sanity
if [ -z "$SINGULARITY_libexecdir" ]; then
    echo "Could not identify the Singularity libexecdir."
    exit 1
fi

## Load functions
if [ -f "$SINGULARITY_libexecdir/singularity/functions" ]; then
    . "$SINGULARITY_libexecdir/singularity/functions"
else
    echo "Error loading functions: $SINGULARITY_libexecdir/singularity/functions"
    exit 1
fi

singularity_import linux_build

SINGULARITY_BUILDDEF="${1:-}"
shift
SINGULARITY_TMPDIR=`mktemp -d /tmp/singularity-bootstrap.XXXXXXX`
PATH=/bin:/sbin:$PATH

export SINGULARITY_TMPDIR SINGULARITY_BUILDDEF

set -e

# Always run these checks
SanityCheck
PreSetup


if [ -n "${SINGULARITY_BUILDDEF:-}" -a -f "$SINGULARITY_BUILDDEF" ]; then
    # sourcing without a leading slash is weird and requires PATH
    PATH=".:$PATH"
    . $SINGULARITY_BUILDDEF
fi

Finalize

rm -rf "$SINGULARITY_TMPDIR"
