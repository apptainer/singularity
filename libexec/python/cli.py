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

from docker.api import get_layer, create_runscript, get_manifest, get_config, get_images
from utils import extract_tar, change_permissions
import argparse
import os
import re
import sys
import tempfile

def main():
    parser = argparse.ArgumentParser(description="bootstrap Docker images for Singularity containers")

    # Name of the docker image, required
    parser.add_argument("--docker", 
                        dest='docker', 
                        help="name of Docker image to bootstrap, in format library/ubuntu:latest", 
                        type=str, 
                        default=None)

    # root file system of singularity image
    parser.add_argument("--rootfs", 
                        dest='rootfs', 
                        help="the path for the root filesystem to extract to", 
                        type=str, 
                        default=None)

    # Docker registry (default is registry-1.docker.io
    parser.add_argument("--registry", 
                        dest='registry', 
                        help="the registry path to use, to replace registry-1.docker.io", 
                        type=str, 
                        default=None)


    # Flag to add the Docker CMD as a runscript
    parser.add_argument("--cmd", 
                        dest='includecmd', 
                        action="store_true",
                        help="boolean to specify that the CMD should be included as a runscript (default is not included)", 
                        default=False)


    # Flag to indicate a token is not required
    parser.add_argument("--no-token", 
                        dest='notoken', 
                        action="store_true",
                        help="boolean to specify that the CMD should be included as a runscript (default is not included)", 
                        default=False)

    
    try:
        args = parser.parse_args()
    except:
        parser.print_help()
        sys.exit(0)

    # Find root filesystem location
    if args.rootfs != None:
       singularity_rootfs = args.rootfs
    else:
       singularity_rootfs = os.environ.get("SINGULARITY_ROOTFS",None)
       if singularity_rootfs == None:
           print("ERROR: root file system not specified or defined as environmental variable, exiting!")
           sys.exit(1)

    # Does the registry require a token?
    doauth = True
    if args.notoken == True:
       doauth = False

    # Does the user want to include the CMD as runscript?
    includecmd = args.includecmd

    # Do we have a docker image specified?
    if args.docker != None:
        image = args.docker



# INPUT PARSING -------------------------------------------
# Parse image name, repo name, and namespace


        # First split the docker image name by /
        image = image.split('/')

        # If there are two parts, we have namespace with repo (and maybe tab)
        if len(image) == 2:
            namespace = image[0]
            image = image[1]

        # Otherwise, we must be using library namespace
        else:
            namespace = "library"
            image = image[0]

        # Now split the docker image name by :
        image = image.split(':')
        if len(image) == 2:
            repo_name = image[0]
            repo_tag = image[1]

        # Otherwise, assume latest of an image
        else:
            repo_name = image[0]
            repo_tag = "latest"

        # Tell the user the namespace, repo name and tag
        print("%s/%s:%s" %(namespace,repo_name,repo_tag))


# IMAGE METADATA -------------------------------------------
# Use Docker Registry API (version 2.0) to get images ids, manifest

        # Get an image manifest - has image ids to parse, and will be
        # used later to get Cmd
        manifest = get_manifest(repo_name=repo_name,
                                namespace=namespace,
                                repo_tag=repo_tag,
                                registry=args.registry,
                                auth=doauth)

        # Get images from manifest using version 2.0 of Docker Registry API
        images = get_images(manifest=manifest,
                            registry=args.registry,
                            auth=doauth)
        
       
#  DOWNLOAD LAYERS -------------------------------------------
# Each is a .tar.gz file, obtained from registry with curl

        # Create a temporary directory for targzs
        tmpdir = tempfile.mkdtemp()
        layers = []

        for image_id in images:

            # Download the layer
            targz = get_layer(image_id=image_id,
                              namespace=namespace,
                              repo_name=repo_name,
                              download_folder=tmpdir,
                              registry=args.registry,
                              auth=doauth) 

            layers.append(targz) # in case we want a list at the end
                                 # @chrisfilo suggestion to try compiling into one tar.gz

            # Extract image and remove tar
            extract_tar(targz,singularity_rootfs)
            os.remove(targz)
               
     
    # If the user wants to include the CMD as runscript, generate it here
    if includecmd == True:

        cmd = get_config(manifest) # default is spec="Cmd"

        # Only add runscript if command is defined
        if cmd != None:
            print("Adding Docker CMD as Singularity runscript...")
            runscript = create_runscript(cmd=cmd,
                                         base_dir=singularity_rootfs)

            # change permission of runscript to 0755 (default)
            change_permissions("%s/singularity" %(singularity_rootfs))

    # When we finish, change permissions for the entire thing
    #change_permissions("%s/" %(singularity_rootfs))


if __name__ == '__main__':
    main()
