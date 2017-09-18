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

test_init "Checking Python unit tests"

cd ../libexec/python

if which python2 >/dev/null 2>&1; then
    stest 0 python2 -m unittest tests.test_json
    stest 0 python2 -m unittest tests.test_helpers
    stest 0 python2 -m unittest tests.test_base
    stest 0 python2 -m unittest tests.test_core
    stest 0 python2 -m unittest tests.test_docker_import
    stest 0 python2 -m unittest tests.test_docker_api
    stest 0 python2 -m unittest tests.test_docker_tasks
    stest 0 python2 -m unittest tests.test_shub_pull
    stest 0 python2 -m unittest tests.test_shub_api
    stest 0 python2 -m unittest tests.test_custom_cache
    stest 0 python2 -m unittest tests.test_default_cache
    stest 0 python2 -m unittest tests.test_disable_cache
else
    echo "Skipping python2 tests: not installed"
fi

if which python3 >/dev/null 2>&1; then
    stest 0 python3 -m unittest tests.test_json
    stest 0 python3 -m unittest tests.test_helpers
    stest 0 python3 -m unittest tests.test_base
    stest 0 python3 -m unittest tests.test_core
    stest 0 python3 -m unittest tests.test_docker_import
    stest 0 python3 -m unittest tests.test_docker_api
    stest 0 python3 -m unittest tests.test_docker_tasks
    stest 0 python3 -m unittest tests.test_shub_pull
    stest 0 python3 -m unittest tests.test_shub_api
    stest 0 python3 -m unittest tests.test_custom_cache
    stest 0 python3 -m unittest tests.test_default_cache
    stest 0 python3 -m unittest tests.test_disable_cache
else
    echo "Skipping python3 tests: not installed"
fi

if which pylint >/dev/null 2>&1; then
    stest 0 pylint $PWD --errors-only --ignore tests  --disable=E0401,E0611,E1101
else
    echo "Skipping pylint tests: not installed"
fi


test_cleanup
