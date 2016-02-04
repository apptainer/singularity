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

TXT_RESOLVERS="perlmods_resolver $TXT_RESOLVERS"


perlmods_resolver() {
    for file in $@; do
        if file "$file" | grep -q "Perl"; then
            egrep "^\s*(use|require|no)\s*" "$file" | sed -e 's/^\s*\S*\s*\(\S*\).*/\1/' | while read req; do
                $libexecdir/singularity/ftrace `which perl` -M"$req" -e 'exit;' 2>&1 >/dev/null | while read FILE; do
                    if [ ! -f "$INSTALLDIR/c/$FILE" ]; then
                        install_file "$FILE"
                    fi
                done
            done
        fi
    done
}

