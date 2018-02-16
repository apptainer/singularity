#!/bin/bash
# 
# Copyright (c) 2017-2018, SyLabs, Inc. All rights reserved.
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

if [ -z "${DAEMON_PID}" ]; then
    message ERROR "No PID found for daemon process\n"
    ABORT 255
fi

# determine is we need to enter user namespace
self_inode=$(stat -L -c "%i" /proc/self/ns/user 2>/dev/null)
daemon_inode=$(stat -L -c "%i" /proc/$DAEMON_PID/ns/user 2>/dev/null)
if [ $? = 0 ]; then
    if [ ! -z "$self_inode" -a ! -z "$daemon_inode" ]; then
        if [ "$self_inode" != "$daemon_inode" ]; then
            SINGULARITY_NOSUID=1
            export SINGULARITY_NOSUID
        fi
    fi
fi

if [ ! -z "${ADD_CAPS:-}" ]; then
    export SINGULARITY_ADD_CAPS="${ADD_CAPS}"
fi

if [ ! -z "${DROP_CAPS:-}" ]; then
    export SINGULARITY_DROP_CAPS="${DROP_CAPS}"
fi

if [ ! -z "${KEEP_PRIVS:-}" ]; then
    export SINGULARITY_KEEP_PRIVS="1"
fi

if [ ! -z "${NO_PRIVS:-}" ]; then
    export SINGULARITY_NO_PRIVS="1"
fi

SINGULARITY_IMAGE="${DAEMON_IMAGE}"
SINGULARITY_DAEMON_JOIN=1
export SINGULARITY_DAEMON_JOIN SINGULARITY_IMAGE

