#!/bin/bash

if [ ! -f "autogen.sh" ]; then
    /bin/echo "ERROR: Run this from the singularity source root"
    exit 1
fi

if [ ! -f "libexec/functions" ]; then
    /bin/echo "ERROR: Could not find functions file"
    exit 1
fi


MESSAGELEVEL=3
TEMPDIR=`mktemp -d /tmp/singularity-test.XXXXXX`
SINGULARITY_CACHEDIR="$TEMPDIR"
export SINGULARITY_CACHEDIR MESSAGELEVEL


/bin/echo "Gaining/checking sudo access..."
sudo true

if [ -z "$CLEAN_SHELL" ]; then
    /bin/echo "Building/Installing Singularity to temporary directory"
    /bin/echo "Reinvoking in a clean shell"
    sleep 1
    exec env -i CLEAN_SHELL=1 PATH="/bin:/usr/bin:/sbin:/usr/sbin" bash "$0" "$*"
fi

. ./libexec/functions

make maintainer-clean >/dev/null 2>&1
stest 0 sh ./autogen.sh --prefix="$TEMPDIR"
stest 0 make
stest 0 make install

PATH="$TEMPDIR/bin:$PATH"

/bin/echo
/bin/echo "SINGULARITY_CACHEDIR=$SINGULARITY_CACHEDIR"
/bin/echo "PATH=$PATH"
/bin/echo

/bin/echo "$Creating temp working space at: $TEMPDIR"
stest 0 mkdir -p "$TEMPDIR"
stest 0 pushd "$TEMPDIR"

/bin/echo
/bin/echo "${BLUE}Running tests...${NORMAL}"
/bin/echo

# Testing singularity internal commands
stest 0 singularity
stest 0 singularity help
stest 0 singularity help shell
stest 0 singularity -h
stest 0 singularity --help
stest 0 singularity --version




/bin/echo
/bin/echo "Cleaning up"
stest 0 popd
stest 0 rm -rf "$TEMPDIR"
stest 0 make maintainer-clean

/bin/echo
/bin/echo "Done. All tests completed successfully"
/bin/echo

