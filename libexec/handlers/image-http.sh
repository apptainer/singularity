#!/bin/bash
# 
# Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
# Copyright (c) 2015-2017, Gregory M. Kurtzer. All rights reserved.


NAME=`basename "$SINGULARITY_IMAGE"`

if [ -f "$NAME" ]; then
    message 2 "Using cached container in current working directory: $NAME\n"
    SINGULARITY_IMAGE="$NAME"
else
    message 1 "Caching container to current working directory: $NAME\n"
    if curl -L -k "$SINGULARITY_IMAGE" > "$NAME"; then
        SINGULARITY_IMAGE="$NAME"
    else
        ABORT 255
    fi
fi

