#!/bin/sh

if autoreconf -V >/dev/null 2>&1 ; then
    set -x
    autoreconf -i -f
else
    set -x
    libtoolize -c
    aclocal
    autoheader
    autoconf
    automake -ca -Wno-portability
fi

if [ -z "$NO_CONFIGURE" ]; then
   ./configure "$@" 
fi

