#!/bin/bash
#
# Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
#
# See the COPYRIGHT.md file at the top-level directory of this distribution and at
# https://github.com/singularityware/singularity/blob/master/COPYRIGHT.md.
#
# This file is part of the Singularity Linux container project. It is subject to the license
# terms in the LICENSE.md file found in the top-level directory of this distribution and
# at https://github.com/singularityware/singularity/blob/master/LICENSE.md. No part
# of Singularity, including this file, may be copied, modified, propagated, or distributed
# except according to the terms contained in the LICENSE.md file.


if test -n "${SINGULARITY_APPNAME:-}"; then

    # The active app should be exported
    export SINGULARITY_APPNAME

    if test -d "/scif/apps/${SINGULARITY_APPNAME:-}/"; then
        SINGULARITY_APPS="/scif/apps"
        SINGULARITY_APPROOT="/scif/apps/${SINGULARITY_APPNAME:-}"
        export SINGULARITY_APPROOT SINGULARITY_APPS
        PATH="/scif/apps/${SINGULARITY_APPNAME:-}:$PATH"

        # Automatically add application bin to path
        if test -d "/scif/apps/${SINGULARITY_APPNAME:-}/bin"; then
            PATH="/scif/apps/${SINGULARITY_APPNAME:-}/bin:$PATH"
        fi

        # Automatically add application lib to LD_LIBRARY_PATH
        if test -d "/scif/apps/${SINGULARITY_APPNAME:-}/lib"; then
            LD_LIBRARY_PATH="/scif/apps/${SINGULARITY_APPNAME:-}/lib:$LD_LIBRARY_PATH"
            export LD_LIBRARY_PATH
        fi

        # Automatically source environment
        if [ -f "/scif/apps/${SINGULARITY_APPNAME:-}/scif/env/01-base.sh" ]; then
            . "/scif/apps/${SINGULARITY_APPNAME:-}/scif/env/01-base.sh"
        fi
        if [ -f "/scif/apps/${SINGULARITY_APPNAME:-}/scif/env/90-environment.sh" ]; then
            . "/scif/apps/${SINGULARITY_APPNAME:-}/scif/env/90-environment.sh"
        fi

        export PATH
    else
        echo "Could not locate the container application: ${SINGULARITY_APPNAME}"
        exit 1
    fi
fi

