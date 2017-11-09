#!/bin/bash
#
# Copyright (c) 2015-2016, Gregory M. Kurtzer. All rights reserved.
#
# "Singularity" Copyright (c) 2016, The Regents of the University of California,
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



. ./functions

test_init "Testing custom deffile options"



CONTAINER="$SINGULARITY_TESTDIR/container.img"
DEFFILE="$SINGULARITY_TESTDIR/container.def"


cat <<EOF > "$DEFFILE"
Bootstrap: docker
From: busybox

%runscript
true
EOF


stest 0 sudo singularity build -F "$CONTAINER" "$DEFFILE"
stest 0 singularity exec "$CONTAINER" true
stest 0 singularity run "$CONTAINER"


cat <<EOF > "$DEFFILE"
Bootstrap: docker
From: busybox

%runscript
false
EOF

stest 0 sudo singularity build -F "$CONTAINER" "$DEFFILE"
stest 0 singularity exec "$CONTAINER" true
stest 1 singularity run "$CONTAINER"


cat <<EOF > "$DEFFILE"
Bootstrap: docker
From: busybox

%files

 # Spaces and comments
$DEFFILE /deffile

../Makefile /makefile

%post
touch /testfile
EOF

stest 0 sudo singularity build -F "$CONTAINER" "$DEFFILE"
stest 0 singularity exec "$CONTAINER" true
stest 0 singularity exec "$CONTAINER" test -f /deffile
stest 0 singularity exec "$CONTAINER" test -f /testfile
stest 0 singularity exec "$CONTAINER" test -f /makefile


cat <<EOF > "$DEFFILE"
Bootstrap: docker
From: busybox

%environment
echo "hi from environment"
EOF

stest 0 sudo singularity build -F "$CONTAINER" "$DEFFILE"
stest 0 singularity exec "$CONTAINER" true
stest 1 singularity exec "$CONTAINER" false
stest 0 sh -c "echo true | singularity shell "$CONTAINER" | grep 'hi from environment'"
stest 0 sh -c "singularity exec "$CONTAINER" true | grep 'hi from environment'"


# Test for #1103: Duplicate bootstrap definition key found: '#UpdateURL'
cat <<EOF > "$DEFFILE"
Bootstrap: docker
From: busybox#comment that needs to be ignored

#DuplicateKey: commented value
#DuplicateKey: other commented value
# also include exact example found in #1103
#UpdateURL: http://mirror.centos.org/centos-%{OSVERSION}/%{OSVERSION}/updates/\$basearch/
#UpdateURL: http://mirror.centos.org/centos-7/7.4.1708/updates/x86_64/
EOF

stest 0 sudo singularity build -F "$CONTAINER" "$DEFFILE"
stest 0 singularity exec "$CONTAINER" true
stest 1 singularity exec "$CONTAINER" false

# Test multiple empty lines
cat <<EOF > "$DEFFILE"
Bootstrap: docker
From: busybox


# this is a comment
EOF

stest 0 sudo singularity build -F "$CONTAINER" "$DEFFILE"
stest 0 singularity exec "$CONTAINER" true
stest 1 singularity exec "$CONTAINER" false

# Test comments inside runscript
cat <<EOF > "$DEFFILE"
Bootstrap: docker
From: busybox

%runscript
#this is a comment
EOF
# expected output of test below
cat <<EOF >"$SINGULARITY_TESTDIR/expected.txt"
#!/bin/sh 

#this is a comment
EOF

stest 0 sudo singularity build -F "$CONTAINER" "$DEFFILE"
stest 0 singularity exec "$CONTAINER" cat /.singularity.d/runscript
# save output of last test, need it to compare with expected
cp "$SINGULARITY_TESTDIR/output" "$SINGULARITY_TESTDIR/actual.txt"
stest 0 cmp "$SINGULARITY_TESTDIR/actual.txt" "$SINGULARITY_TESTDIR/expected.txt"


test_cleanup