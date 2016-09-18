#!/usr/bin/env python

'''
bootstrap.py: python helper for singularity command line tool

'''

from docker import list_images, get_token, get_tags, get_layer
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
                    

    # When we finish, change permissions for the entire thing
    change_permissions("%s/" %(singularity_rootfs))


if __name__ == '__main__':
    main()
