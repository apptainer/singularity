#!/bin/bash
# 
# Copyright (c) 2015-2017, Gregory M. Kurtzer. All rights reserved.
# 
# Copyright (c) 2016-2017, The Regents of the University of California,
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

if [ -z "${SINGULARITY_ROOTFS:-}" ]; then
    message ERROR "Singularity root file system not defined\n"
    exit 1
fi

if [ -z "${SINGULARITY_BUILDDEF:-}" ]; then
    message ERROR "Singularity bootstrap definition file not defined!\n"
    exit 1
fi

if [ ! -f "${SINGULARITY_BUILDDEF:-}" ]; then
    message ERROR "Singularity bootstrap definition file not found!\n"
    exit 1
fi


# First priority goes to runscript defined in build file
runscript_command=$(singularity_section_get "runscript" "$SINGULARITY_BUILDDEF")

# If the command is not empty, write to file.
if [ ! -z "$runscript_command" ]; then
    echo "User defined %runscript found! Taking priority."
    echo "$runscript_command" > "$SINGULARITY_ROOTFS/singularity"    
fi

test -d "$SINGULARITY_ROOTFS/proc" || install -d -m 755 "$SINGULARITY_ROOTFS/proc"
test -d "$SINGULARITY_ROOTFS/sys" || install -d -m 755 "$SINGULARITY_ROOTFS/sys"
test -d "$SINGULARITY_ROOTFS/tmp" || install -d -m 755 "$SINGULARITY_ROOTFS/tmp"
test -d "$SINGULARITY_ROOTFS/dev" || install -d -m 755 "$SINGULARITY_ROOTFS/dev"

mount --no-mtab -t proc proc "$SINGULARITY_ROOTFS/proc"
mount --no-mtab -t sysfs sysfs "$SINGULARITY_ROOTFS/sys"
mount --no-mtab --rbind "/tmp" "$SINGULARITY_ROOTFS/tmp"
mount --no-mtab --rbind "/dev" "$SINGULARITY_ROOTFS/dev"

cp /etc/hosts           "$SINGULARITY_ROOTFS/etc/hosts"
cp /etc/resolv.conf     "$SINGULARITY_ROOTFS/etc/resolv.conf"

### EXPORT ENVARS
DEBIAN_FRONTEND=noninteractive
export DEBIAN_FRONTEND


### RUN SETUP
if singularity_section_exists "setup" "$SINGULARITY_BUILDDEF"; then
    ARGS=`singularity_section_args "setup" "$SINGULARITY_BUILDDEF"`
    singularity_section_get "setup" "$SINGULARITY_BUILDDEF" | /bin/sh -e -x $ARGS || ABORT 255
fi

if [ ! -x "$SINGULARITY_ROOTFS/bin/sh" -a ! -L "$SINGULARITY_ROOTFS/bin/sh" ]; then
    message ERROR "Could not locate /bin/sh inside the container\n"
    exit 255
fi

### RUN POST
if singularity_section_exists "post" "$SINGULARITY_BUILDDEF"; then
    message 1 "Running post scriptlet\n"

    ARGS=`singularity_section_args "post" "$SINGULARITY_BUILDDEF"`
    singularity_section_get "post" "$SINGULARITY_BUILDDEF" | chroot "$SINGULARITY_ROOTFS" /bin/sh -e -x $ARGS || ABORT 255
fi

### ENVIRONMENT
if singularity_section_exists "environment" "$SINGULARITY_BUILDDEF"; then
    message 1 "Adding environment to container\n"

    singularity_section_get "environment" "$SINGULARITY_BUILDDEF" >> "$SINGULARITY_ROOTFS/.singularity.d/env/90-builddef.sh"
fi

### LABELS
if singularity_section_exists "labels" "$SINGULARITY_BUILDDEF"; then
    message 1 "Adding deffile section labels to container\n"

    singularity_section_get "labels" "$SINGULARITY_BUILDDEF" | while read KEY VAL; do
        if [ -n "$KEY" -a -n "$VAL" ]; then
            $SINGULARITY_libexecdir/singularity/python/helpers/json/add.py --key "$KEY" --value "$VAL" --file "$SINGULARITY_ROOTFS/.singularity.d/labels.json"
            set +x
        fi
    done
fi


### FILES
if singularity_section_exists "files" "$SINGULARITY_BUILDDEF"; then
    message 1 "Adding files to container\n"

    singularity_section_get "files" "$SINGULARITY_BUILDDEF" | sed -e 's/#.*//' | while read origin dest; do
        if [ -n "${origin:-}" ]; then
            if [ -z "${dest:-}" ]; then
                dest="$origin"
            fi
            message 1 "Copying '$origin' to '$dest'\n"
            if ! /bin/cp -fLr $origin "$SINGULARITY_ROOTFS/$dest"; then
                message ERROR "Failed copying file(s) into container\n"
                exit 255
            fi
        fi
    done
fi


### RUN TEST
if singularity_section_exists "test" "$SINGULARITY_BUILDDEF"; then
    message 1 "Running test scriptlet\n"

    ARGS=`singularity_section_args "test" "$SINGULARITY_BUILDDEF"`
    echo "#!/bin/sh" > "$SINGULARITY_ROOTFS/.test"
    echo "" >> "$SINGULARITY_ROOTFS/.test"
    singularity_section_get "test" "$SINGULARITY_BUILDDEF" >> "$SINGULARITY_ROOTFS/.test"

    chmod 0755 "$SINGULARITY_ROOTFS/.test"

    chroot "$SINGULARITY_ROOTFS" /bin/sh -e -x $ARGS "/.test" "$@" || ABORT 255
fi

> "$SINGULARITY_ROOTFS/etc/hosts"
> "$SINGULARITY_ROOTFS/etc/resolv.conf"


# If we have a runscript, whether docker, user defined, change permissions
if [ -s "$SINGULARITY_ROOTFS/singularity" ]; then
    chmod 0755 "$SINGULARITY_ROOTFS/singularity"
fi
