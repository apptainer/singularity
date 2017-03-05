#!/usr/bin/env python

'''

add.py: python helper for Singularity ADD, which is an
        import without environment or metadata that returns
        a flat file with docker tars to SINGULARITY_META_DIR
        currently, only supported for Docker, as a singularity
        "add" is equivalent (from the Software function) 
        to a pull.

ENVIRONMENTAL VARIABLES that are found for this executable:

    SINGULARITY_ROOTFS
    SINGULARITY_CONTAINER

    SINGULARITY_DOCKER_USERNAME
    SINGULARITY_DOCKER_PASSWORD
    SINGULARITY_METADATA_BASE

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
from logman import logger
import os
import sys


def main():
    '''main is a wrapper for the client to hand the parser to the executable functions
    This makes it possible to set up a parser in test cases
    '''
    logger.info("\n*** STARTING DOCKER ADD PYTHON  ****")

    from defaults import SINGULARITY_ROOTFS, METADATA_BASE, LAYERFILE
    logger.info("Root file system: %s", SINGULARITY_ROOTFS)
    container = getenv("SINGULARITY_CONTAINER",required=True)
    image_uri = get_image_uri(container)    
    container = remove_image_uri(container)

    # ADD will write list of docker layers to metadata_file
    if image_uri == "docker://":

        from docker.main import ADD
        from utils import basic_auth_header
        username = getenv("SINGULARITY_DOCKER_USERNAME") 
        password = getenv("SINGULARITY_DOCKER_PASSWORD",silent=True)

        auth = None
        if username is not None and password is not None:
            auth = basic_auth_header(username, password)

        manifest = ADD(auth=auth,
                       image=container,
                       layerfile=LAYERFILE)

    else:
        logger.error("uri %s is not a currently supported uri for docker add. Exiting.",image_uri)
        sys.exit(1)


if __name__ == '__main__':
    main()
