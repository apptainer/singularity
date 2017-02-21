#!/usr/bin/env python

'''

import.py: python helper for Singularity docker import


ENVIRONMENTAL VARIABLES that are found for this executable:

    SINGULARITY_CONTAINER 
    SINGULARITY_DOCKER_INCLUDE_CMD 
    SINGULARITY_DOCKER_USERNAME
    SINGULARITY_DOCKER_PASSWORD


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
sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__), os.path.pardir)))
sys.path.append('..')

from main import IMPORT
from shell import get_image_uri
from utils import (
    basic_auth_header,
)

from defaults import getenv
from logman import logger
import os
import sys


def main():
    '''this function will run a docker import, returning a list of layers 
    and environmental variables and metadata to the metadata base
    '''
    from defaults import SINGULARITY_ROOTFS

    logger.info("\n*** STARTING DOCKER IMPORT PYTHON  ****")
    
    container = getenv("SINGULARITY_CONTAINER",required=True)
    includecmd = getenv("SINGULARITY_DOCKER_INCLUDE_CMD")
    username = getenv("SINGULARITY_DOCKER_USERNAME") 
    password = getenv("SINGULARITY_DOCKER_PASSWORD",silent=True)

    # What image is the user asking for?
    image_uri = get_image_uri(container)    
    logger.info("Root file system: %s",SINGULARITY_ROOTFS)

    # Does the registry require authentication?
    auth = None
    if username is not None and password is not None:
        auth = basic_auth_header(username, password)

    if image_uri == "docker://":

        IMPORT(auth=auth,
               image=container,
               rootfs=SINGULARITY_ROOTFS)

    else:
        logger.error("uri %s is not a currently supported uri for docker import. Exiting.",image_uri)
        sys.exit(1)


if __name__ == '__main__':
    main()
