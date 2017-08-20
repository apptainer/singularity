#!/bin/bash
# 
# Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
#
# Copyright (c) 2015-2017, Gregory M. Kurtzer. All rights reserved.
# 
# Copyright (c) 2016-2017, The Regents of the University of California,
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

if test -n "${SINGULARITY_APPNAME:-}"; then
    if test -d "/scif/apps/${SINGULARITY_APPNAME:-}/"; then
        SINGULARITY_APPROOT="/scif/apps/${SINGULARITY_APPNAME:-}"
        export SINGULARITY_APPROOT
        PATH="/scif/apps/${SINGULARITY_APPNAME:-}:$PATH"
        if test -d "/scif/apps/${SINGULARITY_APPNAME:-}/bin"; then
            PATH="/scif/apps/${SINGULARITY_APPNAME:-}/bin:$PATH"
        fi
        if test -d "/scif/apps/${SINGULARITY_APPNAME:-}/lib"; then
            LD_LIBRARY_PATH="/scif/apps/${SINGULARITY_APPNAME:-}/lib:$LD_LIBRARY_PATH"
            export LD_LIBRARY_PATH
        fi
        if test -f "/scif/apps/${SINGULARITY_APPNAME:-}/scif/environment"; then
            . "/scif/apps/${SINGULARITY_APPNAME:-}/scif/environment"
        fi
        export PATH
    else
        echo "Could not locate the container application: ${SINGULARITY_APPNAME}"
        exit 1
    fi
fi

