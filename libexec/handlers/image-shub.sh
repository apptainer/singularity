#!/bin/bash
# 
# Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
# Copyright (c) 2015-2017, Gregory M. Kurtzer. All rights reserved.


if ! SINGULARITY_CONTENTS=`mktemp ${TMPDIR:-/tmp}/.singularity-layerfile.XXXXXX`; then
    message ERROR "Failed to create temporary directory\n"
    ABORT 255
fi

if [ -n "${SINGULARITY_CACHEDIR:-}" ]; then
    SINGULARITY_PULLFOLDER="$SINGULARITY_CACHEDIR"
else
    SINGULARITY_PULLFOLDER="."
fi

SINGULARITY_CONTAINER="$SINGULARITY_IMAGE"
export SINGULARITY_PULLFOLDER SINGULARITY_CONTAINER SINGULARITY_CONTENTS

if ! eval "$SINGULARITY_libexecdir/singularity/python/pull.py"; then
    ABORT 255
fi

# The python script saves names to files in CONTAINER_DIR
SINGULARITY_IMAGE=`cat $SINGULARITY_CONTENTS`
export SINGULARITY_IMAGE

rm -f "$SINGULARITY_CONTENTS"

if [ -f "$SINGULARITY_IMAGE" ]; then
    chmod +x "$SINGULARITY_IMAGE"
else
    message ERROR "Could not locate downloaded image\n"
    ABORT 255
fi
