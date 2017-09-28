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

test_init "Help and usage tests"

alias cmd_check="echo
    echo \"Testing command usage: '\${cmd}'\"
    stest 0 singularity --help \$cmd
    stest 0 singularity -h \$cmd
    stest 0 singularity help \$cmd
    stest 0 singularity \$cmd help
    stest 0 singularity \$cmd -h
    stest 0 singularity \$cmd --help"


MOST_COMMANDS="
    apps
    bootstrap
    build
    check
    create
    exec
    image
    image.create
    image.expand
    image.export
    image.import
    inspect
    mount
    pull
    run
    shell
    test
    instance.start
    instance.list
    instance.stop
"

# Testing singularity internal commands (one word)
stest 0 singularity
stest 0 singularity --help
stest 0 singularity --version

# Testing one word commands
for cmd in $MOST_COMMANDS; do
    cmd_check
done

# Testing two word commands
cmd="image create"
cmd_check
cmd="image expand"
cmd_check
cmd="image export"
cmd_check
cmd="image import"
cmd_check
cmd="instance start"
cmd_check
cmd="instance list"
cmd_check
cmd="instance stop"
cmd_check

/bin/echo
/bin/echo "Testing error on bad commands"

stest 1 singularity help bogus
stest 1 singularity bogus help
stest 1 singularity help instance bogus
stest 1 singularity image bogus help

test_cleanup
