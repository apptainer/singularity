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

TEMPDIR=`mktemp -d /tmp/singularity-test.XXXXXX`
CONTAINER="container.img"
CONTAINERDIR="container_dir"

/bin/echo
/bin/echo "Building test container..."

stest 0 sudo singularity create -s 568 "$CONTAINER"
stest 0 sudo singularity bootstrap "$CONTAINER" "$STARTDIR/examples/busybox.def"

/bin/echo
/bin/echo "Running container shell tests..."

stest 0 singularity shell "$CONTAINER" -c "true"
stest 1 singularity shell "$CONTAINER" -c "false"
stest 0 sh -c "echo true | singularity shell '$CONTAINER'"
stest 1 sh -c "echo false | singularity shell '$CONTAINER'"

/bin/echo
/bin/echo "Running container exec tests..."

stest 0 singularity exec "$CONTAINER" true
stest 0 singularity exec "$CONTAINER" /bin/true
stest 1 singularity exec "$CONTAINER" false
stest 1 singularity exec "$CONTAINER" /bin/false
stest 1 singularity exec "$CONTAINER" /blahh
stest 1 singularity exec "$CONTAINER" blahh
stest 0 sh -c "echo hi | singularity exec $CONTAINER grep hi"
stest 1 sh -c "echo bye | singularity exec $CONTAINER grep hi"

/bin/echo
/bin/echo "Running container run tests..."

# Before we have a runscript, it should invoke a shell
stest 0 singularity run "$CONTAINER" -c true
stest 1 singularity run "$CONTAINER" -c false
echo -ne "#!/bin/sh\n\neval \"\$@\"\n" > singularity
stest 0 chmod 0644 singularity
stest 0 sudo singularity copy "$CONTAINER" -a singularity /
stest 1 singularity run "$CONTAINER" true
stest 0 sudo singularity exec -w "$CONTAINER" chmod 0755 /singularity
stest 0 singularity run "$CONTAINER" true
stest 1 singularity run "$CONTAINER" false
