#!/bin/bash
#
# Copyright (c) 2017, Michael W. Bauer. All rights reserved.
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

stest 0 sudo singularity create -s 568 "$CONTAINER"
stest 0 sudo singularity bootstrap "$CONTAINER" "$STARTDIR/examples/docker.def"
stest 0 sudo rm -rf "$CONTAINER"
stest 0 sudo singularity create -s 568 "$CONTAINER"
stest 0 sh -c "sed "$STARTDIR/examples/busybox.def" -e 's|^MirrorURL:.*|MirrorURL: https://fake_mirror_url|' > "$TEMPDIR/busybox-busted-mirror.def""
stest 255 singularity bootstrap "$CONTAINER" "$TEMPDIR/busybox-busted-mirror.def"
stest 0 sh -c "sudo singularity bootstrap "$CONTAINER" "$TEMPDIR/busybox-busted-mirror.def" 2>&1 | grep -i 'failed fetching mirrorurl'"
stest 0 sudo rm -rf "$CONTAINER"
stest 0 sudo singularity create -s 568 "$CONTAINER"
# We will need a setuid binary (ping) for the NO_NEW_PRIVS test below.
stest 0 sed -i "$STARTDIR/examples/centos.def" -e 's|#InstallPkgs yum vim-minimal|InstallPkgs iputils|'
stest 0 sudo singularity bootstrap "$CONTAINER" "$STARTDIR/examples/busybox.def"


