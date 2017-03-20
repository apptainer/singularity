#!/bin/sh

# set PS4 explicitly to be POSIX shell compatible for set -x
export PS4=+

mkdir m4 >/dev/null 2>&1

if autoreconf -V >/dev/null 2>&1 ; then
    set -x
    autoreconf -i -f
else
    set -x
    libtoolize -c
    aclocal
    autoheader
    autoconf
    automake -ca -Wno-portability --add-missing
fi


