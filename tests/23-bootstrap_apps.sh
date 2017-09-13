#!/bin/bash
#
# Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
# Copyright (c) 2017, Vanessa Sochat. All rights reserved.
#
# See the COPYRIGHT.md file at the top-level directory of this distribution and at
# https://github.com/singularityware/singularity/blob/master/COPYRIGHT.md.
#
# This file is part of the Singularity Linux container project. It is subject to the license
# terms in the LICENSE.md file found in the top-level directory of this distribution and
# at https://github.com/singularityware/singularity/blob/master/LICENSE.md. No part
# of Singularity, including this file, may be copied, modified, propagated, or distributed
# except according to the terms contained in the LICENSE.md file.

. ./functions

test_init "Standard Integration Format (SCI-F) Apps bootstrap tests"


CONTAINER="$SINGULARITY_TESTDIR/container.img"
DEFFILE="$SINGULARITY_TESTDIR/example.def"

# Be consistent to bootstrap from Ubuntu 14.04
stest 0 grep ubuntu:14.04 ../examples/apps/Singularity

# Create the container with apps recipe
stest 0 cp ../examples/apps/Singularity "$DEFFILE"
stest 0 singularity create -F -s 568 "$CONTAINER"
stest 0 sudo singularity bootstrap "$CONTAINER" "$DEFFILE"

# Testing exec command
stest 0 singularity exec "$CONTAINER" true
stest 0 singularity exec "$CONTAINER" /bin/true
stest 1 singularity exec "$CONTAINER" false
stest 1 singularity exec "$CONTAINER" /bin/false

# Testing folder organization
stest 0 singularity exec "$CONTAINER" test -d "/scif"
stest 0 singularity exec "$CONTAINER" test -d "/scif/apps"
stest 0 singularity exec "$CONTAINER" test -d "/scif/data"
stest 0 singularity exec "$CONTAINER" test -d "/scif/apps/foo"
stest 0 singularity exec "$CONTAINER" test -d "/scif/apps/bar"
stest 0 singularity exec "$CONTAINER" test -f "/scif/apps/foo/filefoo.exec"
stest 0 singularity exec "$CONTAINER" test -f "/scif/apps/bar/filebar.exec"
stest 0 singularity exec "$CONTAINER" test -d "/scif/data/foo/output"
stest 0 singularity exec "$CONTAINER" test -d "/scif/data/foo/input"

# Metadata folder
stest 0 singularity exec "$CONTAINER" test -d "/scif/apps/foo/scif"
stest 0 singularity exec "$CONTAINER" test -d "/scif/apps/foo/scif/env"
stest 0 singularity exec "$CONTAINER" test -f "/scif/apps/foo/scif/Singularity"
stest 0 singularity exec "$CONTAINER" test -f "/scif/apps/foo/scif/env/01-base.sh"
stest 0 singularity exec "$CONTAINER" test -f "/scif/apps/foo/scif/labels.json"
stest 0 singularity exec "$CONTAINER" test -f "/scif/apps/foo/scif/runscript"
stest 0 singularity exec "$CONTAINER" test -f "/scif/apps/foo/scif/runscript.help"

# Testing help
stest 0 sh -c "singularity help '$CONTAINER' | grep 'No runscript help is defined for this image.'"
stest 0 sh -c "singularity help --app foo '$CONTAINER' | grep 'This is the help for foo!'"
stest 0 sh -c "singularity help --app bar '$CONTAINER' | grep 'No runscript help is defined for this application.'"

# Testing apps
stest 0 sh -c "singularity apps '$CONTAINER' | grep 'foo'"
stest 0 sh -c "singularity apps '$CONTAINER' | grep 'bar'"

# Testing inspect
stest 0 sh -c "singularity inspect --app foo '$CONTAINER' | grep HELLOTHISIS"
stest 0 sh -c "singularity inspect --app foo '$CONTAINER' | grep foo"

# Testing run
stest 0 sh -c "singularity run --app foo '$CONTAINER' | grep 'RUNNING FOO'"
stest 0 sh -c "singularity run --app bar '$CONTAINER' | grep 'No Singularity runscript for contained app: bar'"

test_cleanup
