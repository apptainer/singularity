#!/usr/bin/env python

'''

import.py: python helper for Singularity docker import


ENVIRONMENTAL VARIABLES that are found for this executable:

    SINGULARITY_DOCKER_IMAGE 
    SINGULARITY_DOCKER_INCLUDE_CMD 
    SINGULARITY_DOCKER_USERNAME
    SINGULARITY_DOCKER_PASSWORD
    SINGULARITY_DISABLE_CACHE
    SINGULARITY_METADATA_FOLDER


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

from docker.main import IMPORT
from defaults import (
    METADATA_BASE
    DISABLE_CACHE
)
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

    logger.info("\n*** STARTING DOCKER IMPORT PYTHON  ****")
    
    container = getenv("SINGULARITY_DOCKER_IMAGE",error_on_none=True)
    rootfs = getenv("SINGULARITY_ROOTFS",error_on_none=True)
    includecmd = getenv("SINGULARITY_DOCKER_INCLUDE_CMD")
    username = getenv("SINGULARITY_DOCKER_USERNAME") 
    password = getenv("SINGULARITY_DOCKER_PASSWORD",silent=True)

    # What image is the user asking for?
    image_uri = get_image_uri(container)    
    logger.info("Root file system: %s",rootfs)

    # Does the registry require authentication?
    auth = None
    if username is not None and password is not None:
        auth = basic_auth_header(username, password)

    if image_uri == "docker://":

        IMPORT(auth=auth,
               image=container,
               metadata_dir=METADATA_BASE,
               rootfs=rootfs,
               disable_cache=DISABLE_CACHE)

    else:
        logger.error("uri %s is not a currently supported uri for docker import. Exiting.",image_uri)
        sys.exit(1)


if __name__ == '__main__':
    main()
