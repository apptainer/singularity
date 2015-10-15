#!/bin/sh

if autoreconf -V >/dev/null 2>&1 ; then
    set -x
    autoreconf -f -i
else
    set -x
    libtoolize -f -c
    aclocal
    autoheader
    autoconf
    automake -ca -Wno-portability
fi

if [ -z "$NO_CONFIGURE" ]; then
   ./configure "$@" 
fi

