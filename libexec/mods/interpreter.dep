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


TXT_RESOLVERS="script_resolver $BIN_RESOLVERS"


script_resolver() {
    for file in $@; do
        INT=`head -n 1 "$file" | grep "^#\!/" | sed 's@#!\([^ ]*\).*@\1@'`
        if [ -f "$INT" -a ! -f "$INSTALLDIR/c/$INT" ]; then
            install_file "$INT"
        fi
    done
}

