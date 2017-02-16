#!/usr/bin/env python

'''

pull.py: wrapper for "pull" for Singularity Hub command line tool.

ENVIRONMENTAL VARIABLES that must be found for this executable:


   SINGULARITY_IMAGE_SHUB: maps to container name: shub://vsoch/singularity-images
   SINGULARITY_ROOTFS: the root file system location
   PULL_FOLDER_SHUB: maps to location to pull folder to


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

from shub.main import PULL

from shell import get_image_uri,
from utils import ( 
    basic_auth_header
    get_env
)

from logman import logger
import optparse
import os
import sys


def main():
    '''main is a wrapper for the client to hand the parser to the executable functions
    This makes it possible to set up a parser in test cases
    '''
    logger.info("\n*** STARTING SINGULARITY HUB PYTHON PULL****")
    
    # What image is the user asking for?
    container = get_env("SINGULARITY_IMAGE_SHUB", error_on_none=True)
    pull_folder = get_env("PULL_FOLDER_SHUB")
    rootfs = get_env("SINGULARITY_ROOTFS", error_on_none=True)

    image_uri = get_image_uri(container)
    
    if image_uri == "shub://"

       PULL(image=container,
            rootfs=rootfs,
            pull_folder=pull_folder)

    else:
        logger.error("uri %s is not currently supported for pull. Exiting.",image_uri)
        sys.exit(1)


if __name__ == '__main__':
    main()
