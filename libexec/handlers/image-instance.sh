#!/bin/bash
# 
# Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
# Copyright (c) 2015-2017, Gregory M. Kurtzer. All rights reserved.



NAME=`echo "$SINGULARITY_IMAGE" | sed -e 's@^instance://@@'`

singularity_daemon_file "${NAME}"

if [ ! -f "${SINGULARITY_DAEMON_FILE}" ]; then
    message ERROR "A daemon process is not running with this name: ${NAME}\n"
    ABORT 255
fi

. "${SINGULARITY_DAEMON_FILE}"

if [ -z "${DAEMON_IMAGE}" ]; then
    message ERROR "Image for daemon is not defined\n"
    ABORT 255
fi

if [ ! -f "${DAEMON_IMAGE}" -a ! -d "${DAEMON_IMAGE}" ]; then
    message ERROR "Image for daemon is not found: ${DAEMON_IMAGE}\n"
    ABORT 255
fi

SINGULARITY_IMAGE="${DAEMON_IMAGE}"
SINGULARITY_DAEMON_JOIN=1
export SINGULARITY_DAEMON_JOIN SINGULARITY_IMAGE

