#!/usr/bin/env python

'''

size.py: python helper for Singularity size

ENVIRONMENTAL VARIABLES that are required for this executable:

    SINGULARITY_CONTAINER
    SINGULARITY_CONTENTS

For Docker, layer sizes are determined from the tarballs, and written
to the SINGULARITY_CONTENTS (contentfile). For Singularity Hub,
the image size is read from the manifest

Copyright (c) 2017, Vanessa Sochat. All rights reserved. 

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

from shell import (
    get_image_uri,
    remove_image_uri
)

from defaults import getenv
from message import bot


def main():
    '''this function will run the main size functions and call shub clients 
    '''

    container = getenv("SINGULARITY_CONTAINER",required=True)
    image_uri = get_image_uri(container)    
    container = remove_image_uri(container)
    from defaults import LAYERFILE

    ##############################################################################
    # Singularity Hub ############################################################
    ##############################################################################

    if image_uri == "shub://":

        bot.debug("\n*** STARTING SINGULARITY HUB SIZE PYTHON  ****")    
        from shub.main import SIZE
        SIZE(image=container,
             contentfile=LAYERFILE)

    elif image_uri == "docker://":

        from sutils import basic_auth_header
        from docker.main import SIZE

        bot.debug("Docker sizes will be written to: %s" %LAYERFILE)
        username = getenv("SINGULARITY_DOCKER_USERNAME") 
        password = getenv("SINGULARITY_DOCKER_PASSWORD",silent=True)

        auth = None
        if username is not None and password is not None:
            auth = basic_auth_header(username, password)

        SIZE(image=container,
             auth=auth,
             contentfile=LAYERFILE)

    else:
        bot.error("uri %s is not a currently supported uri for size. Exiting." %image_uri)
        sys.exit(1)


if __name__ == '__main__':
    main()
