#!/bin/bash
#
# Copyright (c) 2017-2018, SyLabs, Inc. All rights reserved.
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
    . "$SINGULARITY_libexecdir/singularity/functions"
else
    echo "Error loading functions: $SINGULARITY_libexecdir/singularity/functions"
    exit 1
fi

PATH=/sbin:/usr/sbin:${PATH:-}
IMAGE_SIZE="512"

while true; do
    case ${1:-} in
        -h|--help|help)
            exec "$SINGULARITY_libexecdir/singularity/cli/help.exec" "$SINGULARITY_COMMAND"
        ;;
        -s|--size)
            shift
            SINGULARITY_IMAGESIZE="${1:-}"
            export SINGULARITY_IMAGESIZE
            shift
        ;;
        -*)
            message ERROR "Unknown option: ${1:-}\n"
            exit 1
        ;;
        *)
            break
        ;;
    esac
done


if [ -f "$SINGULARITY_libexecdir/singularity/cli/$SINGULARITY_COMMAND.info" ]; then
    . "$SINGULARITY_libexecdir/singularity/cli/$SINGULARITY_COMMAND.info"
else
    message ERROR "Could not find the info file for: $SINGULARITY_COMMAND\n"
    ABORT 255
fi

if [ -z "${1:-}" ]; then
    if [ -n "${USAGE:-}" ]; then
        echo "USAGE: $USAGE"
    else
        echo "To see usage summary type: singularity help $SINGULARITY_COMMAND"
    fi
    exit 0
fi


SINGULARITY_IMAGE="${1:-}"
SINGULARITY_WRITABLE=1
export SINGULARITY_IMAGE SINGULARITY_WRITABLE

if [ -z "$SINGULARITY_IMAGE" ]; then
    message ERROR "You must supply a path to an image to expand\n"
    exit 1
fi
if [ ! -f "$SINGULARITY_IMAGE" ]; then
    message ERROR "Singularity image file not found: $SINGULARITY_IMAGE\n"
    exit 1
fi

IMAGE_TYPE=`eval "$SINGULARITY_libexecdir/singularity/bin/image-type" "${SINGULARITY_IMAGE}" 2>/dev/null`

if [ "$IMAGE_TYPE" != "EXT3" ]; then
    message ERROR "This is not a writable Singularity image format\n"
    ABORT 255
fi

if ! touch "$SINGULARITY_IMAGE" >/dev/null 2>&1; then
    message ERROR "Inappropriate permission to modify: $SINGULARITY_IMAGE\n"
    ABORT 255
fi

DDSTATUS="none"
if dd --help | grep -q 'status=noxfer'; then
    DDSTATUS="noxfer"
fi

message 1 "Expanding image by ${SINGULARITY_IMAGESIZE:-768}MB\n"
if ! dd if=/dev/zero bs=1M count=${SINGULARITY_IMAGESIZE:-768} status=${DDSTATUS} >> "$SINGULARITY_IMAGE"; then
    message ERROR "Failed expanding image!\n"
    exit 1
fi

message 1 "Create loop device\n"
SINGULARITY_LOOP_DEVICE=$(losetup --show -o 31 -f $SINGULARITY_IMAGE 2> /dev/null)
if [ $? -ne 0 ]; then
       exit 1
fi


message 1 "Checking image's file system\n"
if ! /sbin/e2fsck -fy "$SINGULARITY_LOOP_DEVICE"; then
    umount "$SINGULARITY_LOOP_DEVICE"
    exit 1
fi

message 1 "Resizing image's file system\n"
if ! /sbin/resize2fs "$SINGULARITY_LOOP_DEVICE"; then
    umount "$SINGULARITY_LOOP_DEVICE"
    exit 1
fi

#For some reason without this dummy sleep sometimes umount failed for me with "device busy"
sleep 3
message 1 "Unmounting loop device: $SINGULARITY_LOOP_DEVICE\n"
losetup -d  "$SINGULARITY_LOOP_DEVICE"

message 1 "Image is done: $SINGULARITY_IMAGE\n"
exit 0

