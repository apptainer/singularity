#!/usr/bin/env python

'''

bootstrap.py: python helper for Singularity command line tool

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


from docker.main import (
    ADD,
    IMPORT
)

from shub.main import PULL

from shell import get_image_uri
from utils import basic_auth_header

from logman import logger
import optparse
import os
import sys

def get_parser():

    parser = optparse.OptionParser(description='get external container layers to add to Singularity containers',
                                   usage="usage: %prog [options]",
                                   version="%prog 2.2")

    # Name of the docker image, required
    parser.add_option("--container", 
                      dest='container', 
                      help="name of Docker or Singularity Hub container to add, with uri.", 
                      type=str, 
                      default=None)

    # Download folder in case of pull, will be used in preference over cache
    parser.add_option("--pull-folder", 
                      dest='pull_folder', 
                      help="Folder to pull image to (only for shub endpoint)", 
                      type=str, 
                      default=None)

    # root file system of singularity image
    parser.add_option("--rootfs", 
                      dest='rootfs', 
                      help="the path for the root filesystem to extract to", 
                      type=str, 
                      default=None)

    # Docker registry (default is registry-1.docker.io
    parser.add_option("--registry", 
                      dest='registry', 
                      help="the registry path to use, to replace registry-1.docker.io", 
                      type=str, 
                      default=None)

    # Docker registry (default is registry-1.docker.io
    parser.add_option("--layerfile", 
                      dest='layerfile', 
                      help="a text file to write layers urls. If provided, no extraction is done by Python.", 
                      type=str, 
                      default=None)

    # Flag to add the Docker CMD as a runscript
    parser.add_option("--cmd", 
                      dest='includecmd', 
                      action="store_true",
                      help="boolean to specify that CMD should be used instead of ENTRYPOINT as the runscript.", 
                      default=False)

    parser.add_option("--username",
                      dest='username',
                      help="username for registry authentication",
                      default=None)

    parser.add_option("--password",
                      dest='password',
                      help="password for registry authentication",
                      default=None)


    # Flag to disable cache
    parser.add_option("--no-cache", 
                      dest='disable_cache', 
                      action="store_true",
                      help="boolean to specify disabling the cache.", 
                      default=False)

    return parser


def main():
    '''main is a wrapper for the client to hand the parser to the executable functions
    This makes it possible to set up a parser in test cases
    '''
    logger.info("\n*** STARTING PYTHON CLIENT PORTION ****")
    parser = get_parser()
    
    try:
        (args,options) = parser.parse_args()
    except:
        logger.error("Input args to %s improperly set, exiting.", os.path.abspath(__file__))
        parser.print_help()
        sys.exit(0)

    # Give the args to the main executable to run
    run(args)


def run(args):

    # What image is the user asking for?
    image_uri = get_image_uri(args.container)
    
    # Find root filesystem location
    if args.rootfs != None:
       rootfs = args.rootfs
    else:
       rootfs = os.environ.get("SINGULARITY_ROOTFS", None)    
    logger.info("Root file system: %s",rootfs)


    # Does the registry require authentication?
    auth = None
    if args.username is not None and args.password is not None:
        auth = basic_auth_header(args.username, args.password)
        logger.info("Username for registry authentication: %s", args.username)


    ################################################################################
    # Docker image #################################################################
    ################################################################################

    if image_uri == "docker://":

        # Write layers to file
        if args.layerfile != None:

            ADD(auth=auth,
                image=args.container,
                layerfile=args.layerfile,
                registry=args.registry)

        else:

            IMPORT(auth=auth,
                   image=args.container,
                   registry=args.registry,
                   rootfs=rootfs,
                   disable_cache=args.disable_cache)



    ################################################################################
    # Singularity Hub image ########################################################
    ################################################################################

    elif image_uri == "shub://"

       if rootfs == None: 
           logger.error("root file system not specified OR defined as environmental variable, exiting!")
           sys.exit(1)

       PULL(image=args.container,
            rootfs=rootfs,
            pull_folder=args.pull_folder)

    else:
        logger.error("uri %s is not currently supported for python bootstrap. Exiting.",image_uri)
        sys.exit(1)


if __name__ == '__main__':
    main()
