#!/bin/sh
# 
# Copyright (c) 2015, Gregory M. Kurtzer
# All rights reserved.
# 
# Copyright (c) 2015, The Regents of the University of California,
# through Lawrence Berkeley National Laboratory (subject to receipt of
# any required approvals from the U.S. Dept. of Energy).
# All rights reserved.
# 
# 


sanity_check() {
    if [ -z "$SAPPSPEC" ]; then
        message 0 "ERROR: SAPP specfile not given\n"
        exit 1
    fi

    if [ -z "$SAPPFILE" ]; then
        message 0 "ERROR: SAPP output file not passed\n"
        exit 1
    fi

    if [ -z "$INSTALLDIR" ]; then
        message 0 "ERROR: INSTALLDIR not given\n"
        exit 1
    fi

    message 2 "Checking SAPPFILE is writable...\n"
    if ! touch "$SAPPFILE" 2>/dev/null; then
        message 0 "ERROR: Could not create $SAPPFILE\n"
        exit 1
    fi
}


setup_paths() {
    if [ -z "$INSTALLDIR" ]; then
        message 0 "ERROR: INSTALLDIR not given\n"
        exit 1
    fi

    message 1 "Creating paths...\n"

    if [ ! -d "$INSTALLDIR" ]; then
        if ! mkdir -p "$INSTALLDIR" 2>/dev/null; then
            message 0 "ERROR: Could not create temporary directory\n";
            exit 1
        fi
    fi

    if [ ! -d "$INSTALLDIR/c" ]; then
        if ! mkdir -p "$INSTALLDIR/c" 2>/dev/null; then
            message 0 "ERROR: Could not create temporary root directory\n";
            exit 1
        fi
    fi
    return 0
}


build_scriptlet() {
    INSTALLROOT="$INSTALLDIR/c"
    DESTDIR="$INSTALLDIR/c"
    export INSTALLROOT DESTDIR

    message 1 "Running build scriptlet\n"
    ( get_section_from_conf "build" "$SAPPSPEC" || exit 1 ) | /bin/sh -x
    if [ "$?" != "0" ]; then
        message 0 "ERROR: 'build' scriptlet exited non-zero\n"
        exit 1
    fi
}


install_packages() {
    message 1 "Evaluating: packages\n"
    get_section_from_conf "packages" $SAPPSPEC | while read pkg; do
        if [ -z "$pkg" ]; then
            continue
        fi
        if [ -x "/usr/bin/rpm" ]; then
            if rpm -q "$pkg" >/dev/null 2>&1; then
                rpm -ql "$pkg" | while read i; do
                    if ! /bin/sh -c "$libexecdir/singularity/mods/install_file $i"; then
                        message 0 "Error: failed processing file: $i\n"
                        exit 1
                    fi
                done
            else
                message 0 "Package is not installed: $pkg\n"
            fi
        elif [ -x "/usr/bin/dpkg" ]; then
            message 0 "ERROR: We don't support dpkg yet... Coming soon!\n"
            exit 1
        fi
    done
    if [ "$?" -ne "0" ]; then
        exit 1
    fi
}




