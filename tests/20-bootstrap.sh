#!/bin/bash
#
# Copyright (c) 2017, Michael W. Bauer. All rights reserved.
# Copyright (c) 2017, Gregory M. Kurtzer. All rights reserved.
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

test_init "Bootstrap tests"



CONTAINER="$SINGULARITY_TESTDIR/container.img"
CONTAINERDIR="$SINGULARITY_TESTDIR/container.dir"

stest 0 singularity create -s 568 "$CONTAINER"
stest 0 sudo singularity bootstrap "$CONTAINER" "../examples/busybox.def"
stest 0 singularity exec "$CONTAINER" true
stest 1 singularity exec "$CONTAINER" false

stest 0 singularity create -F -s 568 "$CONTAINER"
stest 0 sudo singularity bootstrap "$CONTAINER" docker://busybox
stest 0 singularity exec "$CONTAINER" true
stest 1 singularity exec "$CONTAINER" false

stest 0 mkdir "$CONTAINERDIR"
stest 0 sudo singularity bootstrap "$CONTAINERDIR" "../examples/busybox.def"
stest 0 singularity exec "$CONTAINERDIR" true
stest 1 singularity exec "$CONTAINERDIR" false

stest 0 singularity create -F -s 568 "$CONTAINER"
stest 1 singularity bootstrap "$CONTAINER" "../examples/busybox.def"
stest 1 sudo singularity bootstrap "$CONTAINER" "/path/to/nofile"
stest 1 sudo singularity bootstrap "$CONTAINER" docker://something_that_doesnt_exist_ever

stest 0 sudo rm -rf "$CONTAINERDIR"

test_cleanup








## TEMP - Will come from test.sh export ##
prefix="/usr/local"
exec_prefix="${prefix}"
libexecdir="${exec_prefix}/libexec"
sysconfdir="${prefix}/etc"
localstatedir="${prefix}/var"
bindir="${exec_prefix}/bin"

SINGULARITY_libexecdir="$libexecdir"
SINGULARITY_sysconfdir="$sysconfdir"
SINGULARITY_localstatedir="$localstatedir"
SINGULARITY_PATH="$bindir"
## TEMP ##


STARTDIR=`pwd`
TEMPDIR=`mktemp -d /tmp/singularity-test.XXXXXX`
CONTAINER="container.img"
CONTAINERDIR="container_dir"
SINGULARITY_MESSAGELEVEL=5
export SINGULARITY_MESSAGELEVEL

. ../libexec/functions

/bin/echo
/bin/echo "Running container import/export tests"
/bin/echo

/bin/echo "Creating temp working space at: $TEMPDIR"
stest 0 mkdir -p "$TEMPDIR"
stest 0 pushd "$TEMPDIR"


/bin/echo
/bin/echo "Building test container..."

stest 0 sudo singularity create -s 568 "$CONTAINER"
stest 0 sudo singularity bootstrap "$CONTAINER" "$STARTDIR/../examples/busybox.def"
echo -ne "#!/bin/sh\n\neval \"\$@\"\n" > singularity
stest 0 chmod 0644 singularity
stest 0 sudo singularity copy "$CONTAINER" -a singularity /


/bin/echo
/bin/echo "Checking export/import..."

stest 0 sh -c "singularity export "$CONTAINER" > ${CONTAINERDIR}.tar"
stest 0 sudo chmod 0644 ${CONTAINERDIR}.tar
stest 0 singularity create -F -s 568 "$CONTAINER"
stest 0 sh -c "cat ${CONTAINERDIR}.tar | singularity import $CONTAINER"
stest 1 sh -c "sudo singularity import $CONTAINER not_a_file 2>&1 | grep \"unbound variable\""
stest 0 singularity create -F -s 568 "$CONTAINER"
stest 0 singularity import ${CONTAINER} docker://centos
stest 0 singularity exec ${CONTAINER} test -f /etc/redhat-release
stest 0 singularity create -F -s 568 "$CONTAINER"
stest 1 sudo singularity import ${CONTAINER} http://fake_singularity_url
stest 0 singularity create -F -s 568 "$CONTAINER"
stest 1 sudo singularity import ${CONTAINER} docker://not_a_docker_container/nope_nope_singularity
stest 0 singularity create -F -s 568 "$CONTAINER"
stest 1 sh -c "sudo singularity import ${CONTAINER} docker://not_a_docker_container/nope_nope_singularity | grep 'ERROR: Container does not contain the valid minimum requirement of /bin/sh'"


/bin/echo
/bin/echo "Checking directory mode"

stest 0 mkdir $CONTAINERDIR
stest 0 sudo tar -C $CONTAINERDIR -xvf ${CONTAINERDIR}.tar
stest 0 singularity exec $CONTAINERDIR true
stest 1 singularity exec $CONTAINERDIR false
stest 1 singularity exec /tmp true
stest 1 singularity exec / true


stest 0 popd
stest 0 sudo rm -rf "$TEMPDIR"

/bin/echo
/bin/echo "06-importexport.sh tests OK"
/bin/echo
