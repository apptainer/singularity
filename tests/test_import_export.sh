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

/bin/echo
/bin/echo "Checking export/import..."

stest 0 sudo singularity export -f ${CONTAINERDIR}.tar "$CONTAINER"
stest 0 mkdir $CONTAINERDIR
stest 0 sudo tar -C $CONTAINERDIR -xvf ${CONTAINERDIR}.tar
stest 0 sudo chmod 0644 ${CONTAINERDIR}.tar
stest 0 sudo rm -f "$CONTAINER"
stest 0 sudo singularity create -s 568 "$CONTAINER"
stest 0 sh -c "cat ${CONTAINERDIR}.tar | sudo singularity import $CONTAINER"
stest 1 sh -c "sudo singularity import $CONTAINER not_a_file 2>&1 | grep "unbound variable""
stest 1 sudo singularity import ${CONTAINER} http://fake_singularity_url
stest 1 sudo singularity import ${CONTAINER} docker://not_a_docker_container/nope_nope_singularity
stest 1 sh -c "sudo singularity import ${CONTAINER} docker://not_a_docker_container/nope_nope_singularity | grep 'ERROR: Container does not contain the valid minimum requirement of /bin/sh'"
