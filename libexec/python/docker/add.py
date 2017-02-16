#!/usr/bin/env python

'''

add.py: python helper for Singularity docker add, which is an
        import without environment or metadata that returns
        a flat file with docker tars to SINGULARITY_META_DIR


ENVIRONMENTAL VARIABLES that are found for this executable:

    SINGULARITY_DOCKER_IMAGE 
    SINGULARITY_DOCKER_REGISTRY
    SINGULARITY_DOCKER_USERNAME
    SINGULARITY_DOCKER_PASSWORD
    SINGULARITY_DISABLE_CACHE
    SINGULARITY_METADATA_FILE


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
sys.path.append('..')

from docker.main import ADD
from shell import get_image_uri
from utils import (
    basic_auth_header,
    getenv
)

from logman import logger
import os
import sys


def main():
    '''main is a wrapper for the client to hand the parser to the executable functions
    This makes it possible to set up a parser in test cases
    '''

    logger.info("\n*** STARTING DOCKER ADD PYTHON  ****")
    
    container = getenv("SINGULARITY_DOCKER_IMAGE",error_on_none=True)
    rootfs = getenv("SINGULARITY_ROOTFS",error_on_none=True)
    registry = getenv("SINGULARITY_DOCKER_REGISTRY") 
    username = getenv("SINGULARITY_DOCKER_USERNAME") 
    password = getenv("SINGULARITY_DOCKER_PASSWORD",silent=True)
    disable_cache = getenv("SINGULARITY_DISABLE_CACHE",default=False)
    metadata_file = getenv("SINGULARITY_METADATA_FILE",error_on_none=True)

    # What image is the user asking for?
    image_uri = get_image_uri(container)    
    logger.info("Root file system: %s",rootfs)

    # Does the registry require authentication?
    auth = None
    if username is not None and password is not None:
        auth = basic_auth_header(username, password)

    ################################################################################
    # Docker image #################################################################
    ################################################################################

    if image_uri == "docker://":

        additions = ADD(auth=auth,
                        image=container,
                        layerfile=metadata_file,
                        registry=registry)

    else:
        logger.error("uri %s is not a currently supported uri for docker add. Exiting.",image_uri)
        sys.exit(1)


if __name__ == '__main__':
    main()
