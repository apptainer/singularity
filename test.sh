#!/bin/bash


ALL_COMMANDS="exec run shell start stop bootstrap copy create expand export import mount"

if [ ! -f "autogen.sh" ]; then
    /bin/echo "ERROR: Run this from the singularity source root"
    exit 1
fi

if [ ! -f "libexec/functions" ]; then
    /bin/echo "ERROR: Could not find functions file"
    exit 1
fi


MESSAGELEVEL=3
STARTDIR=`pwd`
TEMPDIR=`mktemp -d /tmp/singularity-test.XXXXXX`
TSTIMG="container.img"
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
stest 0 sudo make install

PATH="$TEMPDIR/bin:$PATH"
MESSAGELEVEL=5

/bin/echo
/bin/echo "SINGULARITY_CACHEDIR=$SINGULARITY_CACHEDIR"
/bin/echo "PATH=$PATH"
/bin/echo

/bin/echo "Creating temp working space at: $TEMPDIR"
stest 0 mkdir -p "$TEMPDIR"
stest 0 pushd "$TEMPDIR"

/bin/echo
/bin/echo "Running base tests..."
/bin/echo

# Testing singularity internal commands
stest 0 singularity
stest 0 singularity --help
stest 0 singularity --version
for i in $ALL_COMMANDS; do
    echo
    echo "Testing command usage: '$i'"
    stest 0 singularity --help "$i"
    stest 0 singularity -h "$i"
    stest 0 singularity help "$i"
    stest 0 singularity $i help
    stest 0 singularity $i -h
    stest 0 singularity $i --help
done
stest 1 singularity help bogus
stest 1 singularity bogus help

/bin/echo
/bin/echo "Building test container..."

stest 0 sudo singularity create -s 568 "$TSTIMG"
stest 0 sudo singularity bootstrap "$TSTIMG" "$STARTDIR/examples/centos.def"

/bin/echo
/bin/echo "Running container tsts..."

stest 0 singularity shell "$TSTIMG" -c "true"
stest 1 singularity shell "$TSTIMG" -c "false"
stest 0 sh -c "echo true | singularity shell '$TSTIMG'"
stest 1 sh -c "echo false | singularity shell '$TSTIMG'"
stest 0 singularity exec "$TSTIMG" true
stest 1 singularity exec "$TSTIMG" false
stest 0 sh -c "echo hi | singularity exec $TSTIMG grep hi"
stest 1 sh -c "echo bye | singularity exec $TSTIMG grep hi"
stest 0 singularity run "$TSTIMG" -c true
stest 1 singularity run "$TSTIMG" -c false

echo -ne "#!/bin/sh\n\neval \"\$@\"\n" > singularity
chmod 0755 singularity
stest 0 sudo singularity copy "$TSTIMG" singularity /

stest 0 singularity run "$TSTIMG" true
stest 1 singularity run "$TSTIMG" false

stest 1 singularity shell -w "$TSTIMG" -c true
stest 1 singularity exec -w "$TSTIMG" true
stest 1 singularity run -w "$TSTIMG" true

stest 0 sudo singularity export -f out.tar "$TSTIMG"
stest 0 mkdir out
stest 0 sudo tar -C out -xvf out.tar

stest 0 sudo rm -f "$TSTIMG"
stest 0 sudo singularity create -s 568 "$TSTIMG"
stest 0 sh -c "cat out.tar | sudo singularity import $TSTIMG"

/bin/echo
/bin/echo "Cleaning up"
stest 0 popd
stest 0 sudo rm -rf "$TEMPDIR"
stest 0 make maintainer-clean

/bin/echo
/bin/echo "Done. All tests completed successfully"
/bin/echo

