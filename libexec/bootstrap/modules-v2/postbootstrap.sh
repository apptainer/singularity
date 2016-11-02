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

install -d -m 0755 "$SINGULARITY_ROOTFS/bin"
install -d -m 0755 "$SINGULARITY_ROOTFS/dev"
install -d -m 0755 "$SINGULARITY_ROOTFS/home"
install -d -m 0755 "$SINGULARITY_ROOTFS/etc"
install -d -m 0750 "$SINGULARITY_ROOTFS/root"
install -d -m 0755 "$SINGULARITY_ROOTFS/proc"
install -d -m 0755 "$SINGULARITY_ROOTFS/sys"
install -d -m 1777 "$SINGULARITY_ROOTFS/tmp"
install -d -m 1777 "$SINGULARITY_ROOTFS/var/tmp"
touch "$SINGULARITY_ROOTFS/etc/hosts"
touch "$SINGULARITY_ROOTFS/etc/resolv.conf"

test -L "$SINGULARITY_ROOTFS/etc/mtab"  && rm -f "$SINGULARITY_ROOTFS/etc/mtab"

cat > "$SINGULARITY_ROOTFS/etc/mtab" << EOF
singularity / rootfs rw 0 0
EOF

if [ ! -f "$SINGULARITY_ROOTFS/environment" ]; then
cat > "$SINGULARITY_ROOTFS/environment" << EOF
# Define any environment init code here

if test -z "\$SINGULARITY_INIT"; then
    PATH=\$PATH:/bin:/sbin:/usr/bin:/usr/sbin:/usr/local/bin:/usr/local/sbin
    PS1="Singularity.\$SINGULARITY_CONTAINER> \$PS1"
    SINGULARITY_INIT=1
    export PATH PS1 SINGULARITY_INIT
fi
EOF
fi
chmod 0644 "$SINGULARITY_ROOTFS/environment"


if [ -x "$SINGULARITY_ROOTFS/bin/bash" ]; then
    HELPER_SHELL="/bin/bash"
else
    HELPER_SHELL="/bin/sh"
fi

cat > "$SINGULARITY_ROOTFS/.shell" << EOF
#!$HELPER_SHELL
. /environment
if test -n "$\SHELL" -a -x "\$SHELL"; then
    exec "\$SHELL" "\$@"
else
    echo "ERROR: Shell does not exist in container: \$SHELL" 1>&2
    echo "ERROR: Using /bin/sh instead..." 1>&2
fi
if test -x /bin/sh; then
    SHELL=/bin/sh
    export SHELL
    exec /bin/sh "\$@"
else
    echo "ERROR: /bin/sh does not exist in container" 1>&2
fi
exit 1
EOF
chmod 0755 "$SINGULARITY_ROOTFS/.shell"



cat > "$SINGULARITY_ROOTFS/.exec" << EOF
#!$HELPER_SHELL
. /environment
exec "\$@"
EOF
chmod 0755 "$SINGULARITY_ROOTFS/.exec"



cat > "$SINGULARITY_ROOTFS/.run" << EOF
#!$HELPER_SHELL
. /environment
if test -x /singularity; then
    exec /singularity "\$@"
else
    echo "No Singularity runscript found, executing /bin/sh"
    exec /bin/sh "\$@"
fi
EOF
chmod 0755 "$SINGULARITY_ROOTFS/.run"

ldconfig -r "$SINGULARITY_ROOTFS" >/dev/null 2>&1

if [ -f "$SINGULARITY_BUILDDEF" ]; then

#    #TODO: This needs lots of love...
#    singularity_section_get "files" "$SINGULARITY_BUILDDEF" | while read source dest; do
#        if [ -z "$source" ]; then
#            continue
#        fi
#        if [ -z "$dest" ]; then
#            dest="$source"
#        fi
#        dirname=`dirname $dest`
#
#        if [ ! -d "$SINGULARITY_ROOTFS/$dirname" ]; then
#            mkdir -p "$SINGULARITY_ROOTFS/$dirname"
#        fi
#
#        message 1 "Installing file: $source -> $dest\n"
#        cp -r "$source" "$SINGULARITY_ROOTFS/$dest"
#    done

    ### CREATE RUNSCRIPT
    singularity_section_get "runscript" "$SINGULARITY_BUILDDEF" > "$SINGULARITY_ROOTFS/singularity"
    if [ -s "$SINGULARITY_ROOTFS/singularity" ]; then
        chmod 0755 "$SINGULARITY_ROOTFS/singularity"
    else
        rm -f "$SINGULARITY_ROOTFS/singularity"
    fi

    mount -t proc proc "$SINGULARITY_ROOTFS/proc"
    mount -t sysfs sysfs "$SINGULARITY_ROOTFS/sys"

    if [ -d "/dev" -a -d "$SINGULARITY_ROOTFS/dev" ]; then
        mkdir -p -m 0755 "$SINGULARITY_ROOTFS/dev"
    fi
    mount --rbind "/dev/" "$SINGULARITY_ROOTFS/dev"

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

