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

# Things that should always exist in a Singularity container
DIRS="/home /tmp /etc /root /dev /proc /sys /var/tmp"
EMPTY_FILES="/etc/mtab /etc/resolv.conf /etc/nsswitch.conf /etc/hosts"
DEVS="/dev/null /dev/zero /dev/random /dev/urandom"
TMP_REAL_FILES="/etc/resolv.conf /etc/hosts"


SanityCheck() {
    return 0
}

Setup() {
    return 0
}

Bootstrap() {
    if [ ! -f "$SINGULARITY_TMPDIR/type" ]; then
        echo "Bootstrap: You must first call 'DistType'!" >&2
        exit 5
    fi

}

InstallPkgs() {
    if [ ! -f "$SINGULARITY_TMPDIR/type" ]; then
        echo "InstallPkgs: You must first call 'DistType'!" >&2
        exit 5
    fi
    return 0
}

Cleanup() {
    if [ ! -f "$SINGULARITY_TMPDIR/type" ]; then
        echo "Cleanup: You must first call 'DistType'!" >&2
        exit 5
    fi
    return 0
}


DistType() {
    TYPE="$1"

    if [ -z "${TYPE:-}" ]; then
        echo "DistType: Requires an argument!" 2>&2
        exit 1
    fi

    if [ -f "$SINGULARITY_libexecdir/singularity/bootstrap/modules-v1/$TYPE.sh" ]; then
        . "$SINGULARITY_libexecdir/singularity/bootstrap/modules-v1/$TYPE.sh"
    else
        echo "DistType: Unrecognized Distribution type: $TYPE" >&2
        exit 255
    fi

    echo "$TYPE" > "$SINGULARITY_TMPDIR/type"

    return 0
}

MirrorURL() {
    MIRROR="${1:-}"
    export MIRROR

    return 0
}

OSVersion() {
    VERSION="${1:-}"
    export VERSION

    return 0
}

InstallFile() {
    SOURCE="${1:-}"
    DEST="${2:-}"

    if [ -z "${SOURCE:-}" ]; then
        echo "InstallFile: Must be called with a source file!" >&2
        return 1
    fi

    if [ ! -e "$SOURCE" ]; then
        echo "InstallFile: No such file or directory ($SOURCE)" >&2
        return 1
    fi

    if [ -z "${DEST:-}" ]; then
        DEST="$SOURCE"
    fi

    DEST_DIR=`dirname "$DEST"`

    if [ ! -d "$SINGULARITY_ROOTFS/$DEST_DIR" ]; then
        mkdir -p "$SINGULARITY_ROOTFS/$DEST_DIR"
    fi

    cp -rap "$SOURCE" "$SINGULARITY_ROOTFS/$DEST"
    return 0
}

PreSetup() {

    install -d -m 0755 "$SINGULARITY_ROOTFS"
    install -d -m 0755 "$SINGULARITY_ROOTFS/dev"

    cp -a /dev/null         "$SINGULARITY_ROOTFS/dev/null"      2>/dev/null || > "$SINGULARITY_ROOTFS/dev/null"
    cp -a /dev/zero         "$SINGULARITY_ROOTFS/dev/zero"      2>/dev/null || > "$SINGULARITY_ROOTFS/dev/zero"
    cp -a /dev/random       "$SINGULARITY_ROOTFS/dev/random"    2>/dev/null || > "$SINGULARITY_ROOTFS/dev/random"
    cp -a /dev/urandom      "$SINGULARITY_ROOTFS/dev/urandom"   2>/dev/null || > "$SINGULARITY_ROOTFS/dev/urandom"

    if [ ! -f "$SINGULARITY_ROOTFS/environment" ]; then
        echo '# Define any environment init code here' > "$SINGULARITY_ROOTFS/environment"
        echo '# ' >> "$SINGULARITY_ROOTFS/environment"
        echo '' >> "$SINGULARITY_ROOTFS/environment"
        echo 'if test -z "$SINGULARITY_INIT"; then' >> "$SINGULARITY_ROOTFS/environment"
        echo "    PATH=\$PATH:/bin:/sbin:/usr/bin:/usr/sbin:/usr/local/bin:/usr/local/sbin" >> "$SINGULARITY_ROOTFS/environment"
        echo '    PS1="Singularity.$SINGULARITY_CONTAINER> $PS1"' >> "$SINGULARITY_ROOTFS/environment"
        echo '    SINGULARITY_INIT=1' >> "$SINGULARITY_ROOTFS/environment"
        echo '    export PATH PS1 SINGULARITY_INIT' >> "$SINGULARITY_ROOTFS/environment"
        echo 'fi' >> "$SINGULARITY_ROOTFS/environment"
    fi
    chmod 0644 "$SINGULARITY_ROOTFS/environment"

    return 0
}

RunScript() {
    if [ -z "${RAN_RUNSCRIPT:-}" ]; then
        echo '#!/bin/sh'    > "$SINGULARITY_ROOTFS/singularity"
        echo                >> "$SINGULARITY_ROOTFS/singularity"

        grep "^RunScript " "$SINGULARITY_BUILDDEF" | sed -e 's/^RunScript //' >> "$SINGULARITY_ROOTFS/singularity"

        chmod +x "$SINGULARITY_ROOTFS/singularity"

        RAN_RUNSCRIPT=1
    fi

    return 0
}


RunCmd() {
    if ! __runcmd "$@"; then
        message ERROR "Aborting...\n"
        exit 1
    fi

    return 0
}


Finalize() {

    # Make sure directories have sane permissions and create if necessary
    install -d -m 0755 "$SINGULARITY_ROOTFS/bin"
    install -d -m 0755 "$SINGULARITY_ROOTFS/home"
    install -d -m 0755 "$SINGULARITY_ROOTFS/etc"
    install -d -m 0750 "$SINGULARITY_ROOTFS/root"
    install -d -m 0755 "$SINGULARITY_ROOTFS/proc"
    install -d -m 0755 "$SINGULARITY_ROOTFS/sys"
    install -d -m 1777 "$SINGULARITY_ROOTFS/tmp"
    install -d -m 1777 "$SINGULARITY_ROOTFS/var/tmp"

    for i in $EMPTY_FILES; do
        if [ ! -f "$SINGULARITY_ROOTFS/$i" ]; then
            DIRNAME=`dirname "$i"`
            if [ -e "$SINGULARITY_ROOTFS/$i" ]; then
                rm -rf "$SINGULARITY_ROOTFS/$i"
            fi
            if [ ! -d "$DIRNAME" ]; then
                mkdir -m 755 -p "$DIRNAME"
            fi
            > "$SINGULARITY_ROOTFS/$i"
        fi
    done

    if [ -L "$SINGULARITY_ROOTFS/etc/mtab" ]; then
        # Just incase it exists and is a link
        rm -f "$SINGULARITY_ROOTFS/etc/mtab"
    fi

    echo "singularity / rootfs rw 0 0" > "$SINGULARITY_ROOTFS/etc/mtab"

    echo '#!/bin/sh' > "$SINGULARITY_ROOTFS/.shell"
    echo '. /environment' >> "$SINGULARITY_ROOTFS/.shell"
    echo 'if test -n "$SHELL" -a -x "$SHELL"; then' >> "$SINGULARITY_ROOTFS/.shell"
    echo '    exec "$SHELL" "$@"' >> "$SINGULARITY_ROOTFS/.shell"
    echo 'else' >> "$SINGULARITY_ROOTFS/.shell"
    echo '    echo "ERROR: Shell does not exist in container: $SHELL" 1>&2' >> "$SINGULARITY_ROOTFS/.shell"
    echo '    echo "ERROR: Using /bin/sh instead..." 1>&2' >> "$SINGULARITY_ROOTFS/.shell"
    echo 'fi' >> "$SINGULARITY_ROOTFS/.shell"
    echo 'if test -x /bin/sh; then' >> "$SINGULARITY_ROOTFS/.shell"
    echo '    SHELL=/bin/sh' >> "$SINGULARITY_ROOTFS/.shell"
    echo '    export SHELL' >> "$SINGULARITY_ROOTFS/.shell"
    echo '    exec /bin/sh "$@"' >> "$SINGULARITY_ROOTFS/.shell"
    echo 'else' >> "$SINGULARITY_ROOTFS/.shell"
    echo '    echo "ERROR: /bin/sh does not exist in container" 1>&2' >> "$SINGULARITY_ROOTFS/.shell"
    echo 'fi' >> "$SINGULARITY_ROOTFS/.shell"
    echo 'exit 1' >> "$SINGULARITY_ROOTFS/.shell"
    chmod 0755 "$SINGULARITY_ROOTFS/.shell"

    echo '#!/bin/sh' > "$SINGULARITY_ROOTFS/.exec"
    echo '. /environment' >> "$SINGULARITY_ROOTFS/.exec"
    echo 'exec "$@"' >> "$SINGULARITY_ROOTFS/.exec"
    chmod 0755 "$SINGULARITY_ROOTFS/.exec"

    echo '#!/bin/sh' > "$SINGULARITY_ROOTFS/.run"
    echo '. /environment' >> "$SINGULARITY_ROOTFS/.run"
    echo 'if test -x /singularity; then' >> "$SINGULARITY_ROOTFS/.run"
    echo '    exec /singularity "$@"' >> "$SINGULARITY_ROOTFS/.run"
    echo 'else' >> "$SINGULARITY_ROOTFS/.run"
    echo '    echo "No runscript found, executing /bin/sh"' >> "$SINGULARITY_ROOTFS/.run"
    echo '    exec /bin/sh "$@"' >> "$SINGULARITY_ROOTFS/.run"
    echo 'fi' >> "$SINGULARITY_ROOTFS/.run"
    chmod 0755 "$SINGULARITY_ROOTFS/.run"

    return 0
}


__runcmd() {
    CMD="${1:-}"
    shift

    echo "+ $CMD $*" 1>&2

    # Running command through /usr/bin/env -i to sanitize the environment
    chroot "$SINGULARITY_ROOTFS" /usr/bin/env -i PATH="$PATH" "$CMD" "$@"

    return $?
}


__mountproc() {
    if [ -d "/proc" -a -d "$SINGULARITY_ROOTFS/proc" ]; then
        mkdir -p -m 0755 "$SINGULARITY_ROOTFS/proc"
    fi
    mount -t proc proc "$SINGULARITY_ROOTFS/proc"

    return $?
}

__mountsys() {
    if [ ! -d "$SINGULARITY_ROOTFS/sys" ]; then
        mkdir -p -m 0755 "$SINGULARITY_ROOTFS/sys"
    fi
    mount -t sysfs sysfs "$SINGULARITY_ROOTFS/sys"

    return $?
}

__mountdev() {
    if [ -d "/dev" -a -d "$SINGULARITY_ROOTFS/dev" ]; then
        mkdir -p -m 0755 "$SINGULARITY_ROOTFS/dev"
    fi
    mount --rbind "/dev/" "$SINGULARITY_ROOTFS/dev"

    return $?
}


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

