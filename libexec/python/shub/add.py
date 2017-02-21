#!/usr/bin/env python

'''

add.py: python helper for Singularity Hub add, which is basically
        a pull to the SINGULARITY_ROOTFS instead of a user specified
        pull folder


ENVIRONMENTAL VARIABLES that are found for this executable:

    SINGULARITY_ROOTFS
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
import os
sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__), os.path.pardir)))
sys.path.append('..')

from shub.main import ADD
from shell import get_image_uri
from defaults import getenv
from logman import logger
import os
import sys


def main():
    '''main will download a shub image to the METADATA_BASE folder
    '''
    from defaults import LAYERFILE

    container = getenv("SINGULARITY_CONTAINER",required=True)

    logger.info("\n*** STARTING DOCKER ADD PYTHON  ****")

    # What image is the user asking for?
    image_uri = get_image_uri(container)    

    # ADD will write list of docker layers to metadata_file
    if image_uri == "shub://":

        additions = ADD(image=container,
                        layerfile=LAYERFILE)

    else:
        logger.error("uri %s is not a currently supported uri for singularity hub add. Exiting.",image_uri)
        sys.exit(1)


if __name__ == '__main__':
    main()
