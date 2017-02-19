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


# At this point, the container should be valid, and valid i defined by the
# existance of /bin/sh
if [ ! -L "$SINGULARITY_ROOTFS/bin/sh" -a ! -x "$SINGULARITY_ROOTFS/bin/sh" ]; then
    message ERROR "Container does not contain the valid minimum requirement of /bin/sh\n"
    exit 1
fi


if [ -n "${SINGULARITY_BUILDDEF:-}" -a -f "${SINGULARITY_BUILDDEF:-}" ]; then
    # First priority goes to runscript defined in build file
    runscript_command=$(singularity_section_get "runscript" "$SINGULARITY_BUILDDEF")

    # If the command is not empty, write to file.
    if [ ! -z "$runscript_command" ]; then
        echo "User defined %runscript found! Taking priority."
        echo "$runscript_command" > "$SINGULARITY_ROOTFS/singularity"    
    fi

    mount --no-mtab -t proc proc "$SINGULARITY_ROOTFS/proc"
    mount --no-mtab -t sysfs sysfs "$SINGULARITY_ROOTFS/sys"

    if [ -d "/dev" -a -d "$SINGULARITY_ROOTFS/dev" ]; then
        mkdir -p -m 0755 "$SINGULARITY_ROOTFS/dev"
    fi
    mount --no-mtab --rbind "/dev/" "$SINGULARITY_ROOTFS/dev"

    cp /etc/hosts           "$SINGULARITY_ROOTFS/etc/hosts"
    cp /etc/resolv.conf     "$SINGULARITY_ROOTFS/etc/resolv.conf"

    ### RUN SETUP
    if singularity_section_exists "setup" "$SINGULARITY_BUILDDEF"; then
        ARGS=`singularity_section_args "setup" "$SINGULARITY_BUILDDEF"`
        singularity_section_get "setup" "$SINGULARITY_BUILDDEF" | /bin/sh -e -x $ARGS || ABORT 255
    fi

    ### RUN POST
    if singularity_section_exists "post" "$SINGULARITY_BUILDDEF"; then
        if [ "$UID" == "0" ]; then
            if [ -x "$SINGULARITY_ROOTFS/bin/sh" ]; then
                ARGS=`singularity_section_args "post" "$SINGULARITY_BUILDDEF"`
                singularity_section_get "post" "$SINGULARITY_BUILDDEF" | chroot "$SINGULARITY_ROOTFS" /bin/sh -e -x $ARGS || ABORT 255

            else
                message ERROR "Could not run post scriptlet, /bin/sh not found in container\n"
                exit 255
            fi
        else
            message 1 "Not running post scriptlet, not root user\n"
        fi
    fi

    ### RUN TEST
    if singularity_section_exists "test" "$SINGULARITY_BUILDDEF"; then
        if [ "$UID" == "0" ]; then
            if [ -x "$SINGULARITY_ROOTFS/bin/sh" ]; then
                ARGS=`singularity_section_args "test" "$SINGULARITY_BUILDDEF"`
                echo "#!/bin/sh" > "$SINGULARITY_ROOTFS/.test"
                echo "" >> "$SINGULARITY_ROOTFS/.test"
                singularity_section_get "test" "$SINGULARITY_BUILDDEF" >> "$SINGULARITY_ROOTFS/.test"

                chmod 0755 "$SINGULARITY_ROOTFS/.test"

                chroot "$SINGULARITY_ROOTFS" /bin/sh -e -x $ARGS "/.test" "$@" || ABORT 255
            else
                message ERROR "Could not run test scriptlet, /bin/sh not found in container\n"
                exit 255
            fi
        else
            message 1 "Not running test scriptlet, not root user\n"
        fi
    fi

    > "$SINGULARITY_ROOTFS/etc/hosts"
    > "$SINGULARITY_ROOTFS/etc/resolv.conf"

fi

# If we have a runscript, whether docker, user defined, change permissions
if [ -s "$SINGULARITY_ROOTFS/.singularity/runscript" ]; then
    chmod 0755 "$SINGULARITY_ROOTFS/.singularity/runscript"
fi

