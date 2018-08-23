#!/bin/bash
# 
# Copyright (c) 2017-2018, SyLabs, Inc. All rights reserved.
# Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
# Copyright (c) 2015-2017, Gregory M. Kurtzer. All rights reserved.


NAME=`basename "$SINGULARITY_IMAGE"`

CACHE="${SINGULARITY_CACHEDIR:-${HOME}/.singularity}/image_cache"

if [ ! -d ${CACHE} ]; then
    mkdir -p ${CACHE};
fi

if [ -f "${CACHE}/${NAME}" ]; then
    message 2 "Using cached container from: ${CACHE}/${NAME}\n"
    SINGULARITY_IMAGE="${CACHE}/${NAME}"
else
    message 1 "Caching container to: ${CACHE}/${NAME}\n"
    if curl -L -k "$SINGULARITY_IMAGE" > "${CACHE}/${NAME}"; then
        SINGULARITY_IMAGE="${CACHE}/${NAME}"
    else
        ABORT 255
    fi
fi

