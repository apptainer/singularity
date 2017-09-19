#!/usr/bin/env python

'''

pull.py: general "pull" wrapper for Singularity Hub command line tool.
         Currently, only supported endpoint is shub://

ENVIRONMENTAL VARIABLES that are found for this executable:


   SINGULARITY_CONTAINER: container name: shub://vsoch/singularity-images
   SINGULARITY_PULLFOLDER: location to pull image to
   SINGULARITY_METADATA_DIR: if defined, write paths to files here


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

from shell import (
    get_image_uri,
    remove_image_uri
)

from message import bot
import sys


def main():
    '''main is a wrapper for the client to hand the parser
    to the executable functions
    This makes it possible to set up a parser in test cases
    '''
    bot.debug("\n*** STARTING SINGULARITY PYTHON PULL ****")
    from defaults import LAYERFILE, DISABLE_CACHE, getenv

    # What image is the user asking for?
    container = getenv("SINGULARITY_CONTAINER", required=True)
    pull_folder = getenv("SINGULARITY_PULLFOLDER")

    image_uri = get_image_uri(container)
    container = remove_image_uri(container, quiet=True)

    if image_uri == "shub://":

        from shub.main import PULL
        manifest = PULL(image=container,
                        download_folder=pull_folder,
                        layerfile=LAYERFILE)

    else:
        bot.error("uri %s is not supported for pull. Exiting." % (image_uri))
        sys.exit(1)


if __name__ == '__main__':
    main()
