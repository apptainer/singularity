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

from docker import list_images, get_token, get_tags, get_layer, \
    create_runscript, get_manifest 
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

    # Flag to add the Docker CMD as a runscript
    parser.add_argument("--cmd", 
                        dest='includecmd', 
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
# Use Docker Registry API (version 1.0) to get images ids, manifest


        # Get full list of images for the repo
        images = list_images(repo_name=repo_name,
                             namespace=namespace)

        # Get specific image names (first 8 characters) for the tag of interest
        tags = get_tags(repo_name=repo_name,
                        repo_tag=repo_tag,
                        namespace=namespace)

        # Get image manifest? Meta data should be added to image somewhere...

        # Get token (default returns header object)
        token = get_token(repo_name=repo_name,
                          scope="repositories",
                          namespace=namespace,
                          content="images")

        
       
#  DOWNLOAD LAYERS -------------------------------------------
# Each is a .tar.gz file, obtained from registry with curl

        # Create a temporary directory for targzs
        tmpdir = tempfile.mkdtemp()
        layers = []

        for tag in tags:
            image_id = tag['id']

            # Find the corresponding (complete) image id in the images
            match = [x['id'] for x in images if re.search('^%s*' %(image_id),x['id'])]
            if len(match) > 0:
                image_id = match[0]
            else:
                print("WARNING: could not find layer with id %s in Docker registry!" %(image_id))
                continue

            # Download the layer
            targz = get_layer(image_id,token,download_folder=tmpdir) 
            layers.append(targz) # in case we want a list at the end
                                 # @chrisfilo suggestion to try compiling into one tar.gz

            # Extract image
            extract_tar(targz,singularity_rootfs)
                    
    # If the user wants to include the CMD as runscript, generate it here
    if includecmd == True:

        print("Adding Docker CMD as Singularity runscript...")
        manifest = get_manifest(image_id,token)
        cmd = manifest['container_config']['Cmd']

        # Only add runscript if command is defined
        if cmd != None:
            runscript = create_runscript(cmd=cmd,
                                         base_dir=singularity_rootfs)

            # change permission of runscript to 0755 (default)
            change_permissions("%s/runscript" %(singularity_rootfs))
        else:
            print("No Docker CMD found, skipping runscript.")

    # When we finish, change permissions for the entire thing
    #change_permissions("%s/" %(singularity_rootfs))


if __name__ == '__main__':
    main()
