#!/usr/bin/env python

'''

bootstrap.py: python helper for Singularity command line tool

Copyright (c) 2016, Vanessa Sochat. All rights reserved. 

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

from shub.api import (
    download_image, 
    get_manifest as get_shub_manifest,
    get_image_name
)

from docker.api import (
    create_runscript, 
    get_config, 
    get_images,
    get_layer, 
    get_token,
    get_manifest 
)

from shell import parse_image_uri

from utils import (
    basic_auth_header,
    change_permissions, 
    extract_tar, 
    get_cache, 
    is_number,
    write_file
)

from logman import logger
import optparse
import os
import re
import shutil
import sys
import tempfile

def get_parser():


    parser = optparse.OptionParser(description='bootstrap Docker images for Singularity containers',
                                   usage="usage: %prog [options]",
                                   version="%prog 2.2")


    # Name of the docker image, required
    parser.add_option("--docker", 
                      dest='docker', 
                      help="name of Docker image to bootstrap, in format library/ubuntu:latest", 
                      type=str, 
                      default=None)

    # ID of the Singularity Hub container
    parser.add_option("--shub", 
                      dest='shub', 
                      help="unique id or name of the Singularity Hub image", 
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

    # Find root filesystem location
    if args.rootfs != None:
       singularity_rootfs = args.rootfs
    else:
       singularity_rootfs = os.environ.get("SINGULARITY_ROOTFS", None)
       if singularity_rootfs == None and args.shub == None: 
           logger.error("root file system not specified OR defined as environmental variable, exiting!")
           sys.exit(1)
    
    if singularity_rootfs != None:
        logger.info("Root file system defined as %s", singularity_rootfs)

    # Does the registry require authentication?
    auth = None
    if args.username is not None and args.password is not None:
        auth = basic_auth_header(args.username, args.password)
        logger.info("Username for registry authentication: %s", args.username)


    # Does the user want to download a Singularity image?
    if args.shub != None:
        image = args.shub
        manifest = get_shub_manifest(image)
        if args.pull_folder == None:
            cache_base = get_cache(subfolder="shub", 
                                   disable_cache = args.disable_cache)
        else:
            cache_base = args.pull_folder

        # The image name is the md5 hash, download if it's not there
        image_name = get_image_name(manifest)
        image_file = "%s/%s" %(cache_base,image_name)
        if not os.path.exists(image_file):
            image_file = download_image(manifest=manifest,
                                        download_folder=cache_base)
        else:
            print("Image already exists at %s, skipping download." %image_file)
        logger.info("Singularity Hub Image Download: %s", image_file)
       
        # If singularity_rootfs is provided, write metadata to it
        if singularity_rootfs != None:
            logger.debug("Writing SINGULARITY_RUNDIR and SINGULARITY_IMAGE to %s",singularity_rootfs)
            write_file("%s/SINGULARITY_RUNDIR" %singularity_rootfs, os.path.dirname(image_file))
            write_file("%s/SINGULARITY_IMAGE" %singularity_rootfs, image_file)

    # Do we have a docker image specified?
    elif args.docker != None:

        # Does the user want to override default Entrypoint and use CMD as runscript?
        includecmd = args.includecmd
        logger.info("Including Docker command as Runscript? %s", includecmd)

        image = args.docker
        logger.info("Docker image: %s", image)

        # Input Parsing ----------------------------
        # Parse image name, repo name, and namespace

        image = parse_image_uri(image=image,uri="docker://")
        namespace = image['namespace']
        repo_name = image['repo_name']
        repo_tag = image['repo_tag']

        # Tell the user the namespace, repo name and tag
        logger.info("Docker image path: %s/%s:%s", namespace,repo_name,repo_tag)


        # IMAGE METADATA -------------------------------------------
        # Use Docker Registry API (version 2.0) to get images ids, manifest

        # Get an image manifest - has image ids to parse, and will be
        # used later to get Cmd
        manifest = get_manifest(repo_name=repo_name,
                                namespace=namespace,
                                repo_tag=repo_tag,
                                registry=args.registry,
                                auth=auth)

        # Get images from manifest using version 2.0 of Docker Registry API
        images = get_images(manifest=manifest)
       
        #  DOWNLOAD LAYERS -------------------------------------------
        # Each is a .tar.gz file, obtained from registry with curl

        # Get the cache (or temporary one) for docker
        cache_base = get_cache(subfolder="docker", 
                               disable_cache = args.disable_cache)

        layers = []
        for image_id in images:

            # Download the layer, if we don't have it
            targz = "%s/%s.tar.gz" %(cache_base,image_id)
 
            if not os.path.exists(targz):
                targz = get_layer(image_id=image_id,
                                  namespace=namespace,
                                  repo_name=repo_name,
                                  download_folder=cache_base,
                                  registry=args.registry,
                                  auth=auth)

            layers.append(targz) # in case we want a list at the end

            # Extract image and remove tar
            output = extract_tar(targz,singularity_rootfs)
            if output is None:
                logger.error("Error extracting image: %s", targz)
                sys.exit(1)
            if args.disable_cache == True:
                os.remove(targz)
               
     
        # If the user wants to include the CMD as runscript, generate it here
        if includecmd == True:
            spec="Cmd"
        else:
            spec="Entrypoint"

        cmd = get_config(manifest,spec=spec)

        # Only add runscript if command is defined
        if cmd != None:
            print("Adding Docker %s as Singularity runscript..." %(spec.upper()))
            print(cmd)
            runscript = create_runscript(cmd=cmd,
                                         base_dir=singularity_rootfs)

        # When we finish, clean up images
        if args.disable_cache == True:
            shutil.rmtree(cache_base)


        logger.info("*** FINISHING DOCKER BOOTSTRAP PYTHON PORTION ****\n")


if __name__ == '__main__':
    main()
