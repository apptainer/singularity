#!/usr/bin/env python

'''

import.py: python helper for Singularity import


ENVIRONMENTAL VARIABLES that are required for this executable:

    SINGULARITY_CONTAINER
    SINGULARITY_CONTENTS


Given that SINGULARITY_ROOTFS is defined, a full import is done that includes
environment, labels, and extraction of layers. If SINGULARITY_ROOTFS is not
defined, then SINGULARITY_CONTENTS must be defined, which returns a list
of layer contents.

Copyright (c) 2016-2017, Vanessa Sochat. All rights reserved.

"Singularity" Copyright (c) 2016, The Regents of the University of California,
through Lawrence Berkeley National Laboratory (subject to receipt of any
required approvals from the U.S. Dept. of Energy).  All rights reserved.

This software is licensed under a customized 3-clause BSD license.  Please
consult LICENSE file distributed with the sources of this project regarding
your rights to use or distribute this software.

NOTICE.  This Software was developed under funding from the U.S. Department of
Energy and the U.S. Government consequently retains certain rights. As such,
the U.S. Government has been granted for itself and others acting on its
behalf a paid-up, nonexclusive, irrevocable, worldwide license in the Software
to reproduce, distribute copies to the public, prepare derivative works, and
perform publicly and display publicly, and to permit other to do so.


'''

import sys
import os

from shell import (
    get_image_uri,
    remove_image_uri
)

from defaults import getenv
from message import bot


def main():
    '''this function will run a docker import, returning a list of layers
    and environmental variables and metadata to the metadata base
    '''

    container = getenv("SINGULARITY_CONTAINER", required=True)
    image_uri = get_image_uri(container, quiet=True)
    container = remove_image_uri(container)

    ########################################################################
    # Docker Image #########################################################
    ########################################################################

    if image_uri == "docker://":

        bot.debug("\n*** STARTING DOCKER IMPORT PYTHON  ****")

        from sutils import basic_auth_header
        from defaults import LAYERFILE

        bot.debug("Docker layers and metadata will be written to: %s"
                  % (LAYERFILE))

        username = getenv("SINGULARITY_DOCKER_USERNAME")
        password = getenv("SINGULARITY_DOCKER_PASSWORD",
                          silent=True)

        auth = None
        if username is not None and password is not None:
            auth = basic_auth_header(username, password)

        from docker.main import IMPORT

        manifest = IMPORT(auth=auth,
                          image=container,
                          layerfile=LAYERFILE)

    ########################################################################
    # Singularity Hub ######################################################
    ########################################################################

    elif image_uri == "shub://":

        bot.debug("\n*** STARTING SINGULARITY HUB IMPORT PYTHON  ****")

        from defaults import LAYERFILE, LABELFILE
        from shub.main import IMPORT
        IMPORT(image=container,
               layerfile=LAYERFILE,
               labelfile=LABELFILE)

    else:
        bot.error("uri %s is not supported for import. Exiting."
                  % (image_uri))
        sys.exit(1)


if __name__ == '__main__':
    main()
