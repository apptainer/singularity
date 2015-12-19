#!/bin/sh

if [ ! -f "autogen.sh" ]; then
    echo "ERROR: Run this from the singularity source root"
    exit 1
fi

if [ ! -f "libexec/functions" ]; then
    echo "ERROR: Could not find functions file"
    exit 1
fi

MESSAGELEVEL=2
TEMPDIR=`mktemp -d /tmp/singularity-test.XXXXXX`
SINGULARITY_CACHEDIR="$TEMPDIR"
export SINGULARITY_CACHEDIR MESSAGELEVEL

. ./libexec/functions



echo "Gaining/checking sudo access..."
sudo true

echo "Building/Installing Singularity to temporary directory"
exe 0 sh ./autogen.sh --prefix="$TEMPDIR"
exe 0 make
exe 0 make install
exe 0 sudo make install-perms

PATH="$TEMPDIR/bin:$PATH"

echo "Creating temp working space at: $TEMPDIR/tmp"
exe 0 mkdir -p "$TEMPDIR/tmp"
exe 0 pushd "$TEMPDIR/tmp"

echo "Running tests..."
cat <<EOF > example.sspec
Name: cat
Exec: /bin/cat
#DebugOS: 1
EOF
exe 0 singularity --quiet build example.sspec
exe 0 ./cat.sapp example.sspec
echo "Testing pipeline"
if ! cat example.sspec | ./cat.sapp | grep -q '^Name'; then
    echo "failed: cat example.sspec | ./cat.sapp | grep -q '^Name'"
    exit 1
fi
exe 1 singularity list
exe 0 singularity install cat.sapp
exe 0 singularity list
exe 0 singularity check cat
exe 0 singularity run cat example.sspec
exe 0 singularity run cat /etc/hosts
exe 0 singularity run cat /etc/resolv.conf
exe 1 singularity run cat /etc/passwd
exe 1 singularity run cat /etc/shadow
echo "Testing pipeline"
if ! cat example.sspec | singularity run cat | grep -q '^Name'; then
    echo "failed: cat example.sspec | singularity run cat | grep -q '^Name'"
    exit 1
fi
exe 0 singularity strace cat example.sspec
exe 0 singularity test cat
echo "echo hello world" | exe 0 singularity shell cat
exe 0 singularity delete cat
exe 1 singularity list

echo "Testing base Singularity functions"
exe 0 singularity
exe 0 singularity help
exe 0 singularity help build
exe 0 singularity help check
exe 0 singularity help delete
exe 0 singularity help help
exe 0 singularity help install
exe 0 singularity help list
exe 0 singularity help run
exe 0 singularity help shell
exe 0 singularity help specgen
exe 0 singularity help strace
exe 1 singularity help blahblah
exe 0 singularity -h
exe 0 singularity --help
exe 0 singularity --version




echo "Cleaning up"
exe 0 popd
exe 0 rm -rf "$TEMPDIR/tmp"
exe 0 make maintainer-clean

echo
echo "Done. All tests completed successfully"
echo
