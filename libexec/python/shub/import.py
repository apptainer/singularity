#!/usr/bin/env python

'''

import.py: python helper for Singularity docker import


ENVIRONMENTAL VARIABLES that are found for this executable:

    SINGULARITY_CONTAINER

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

from shub.main import IMPORT
from shell import get_image_uri
from defaults import getenv
from logman import logger
import os
import sys


def main():
    '''this function will run a docker import, returning a list of layers 
    and environmental variables and metadata to the metadata base
    '''
    from defaults import LAYERFILE

    logger.info("\n*** STARTING SINGULARITY HUB IMPORT PYTHON  ****")    
    container = getenv("SINGULARITY_CONTAINER",required=True)

    # What image is the user asking for?
    image_uri = get_image_uri(container)    

    if image_uri == "shub://":

        additions = IMPORT(image=container,
                           layerfile=LAYERFILE)

    else:
        logger.error("uri %s is not a currently supported uri for singularity hub import. Exiting.",image_uri)
        sys.exit(1)


if __name__ == '__main__':
    main()
