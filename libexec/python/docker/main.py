'''

main.py: Docker helper functions for Singularity in Python

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
sys.path.append('..') # parent directory

from utils import (
    api_get, 
    extract_tar,
    get_cache, 
    write_file
)

from shell import parse_image_uri
from docker.api import (
    create_runscript,
    extract_env,
    extract_labels,
    get_layer,
    get_manifest
)

from defaults import LAYERFILE

from logman import logger
import json
import re
import os
import tempfile


def IMPORT(image,rootfs,auth=None,disable_cache=False,metadata_dir=None,includecmd=False):
    '''run is the main script that will obtain docker layers, runscript information (either entrypoint
    or cmd), and environment, and either return the list of files to extract (in case of add 
    :param image: the docker image to add
    :param auth: if needed, an authentication header (default None)
    :param metadata_dir: the directory to write metadata, including environment
    :param disable_cache: if True, don't cache layers (default False)
    :param includecmd: use CMD as %runscript instead of ENTRYPOINT (default False)
    '''
    logger.debug("Starting Docker IMPORT, includes environment, runscript, and metadata.")
    logger.info("Docker image: %s", image)

    # Does the user want to override default of using ENTRYPOINT?
    if includecmd:
        logger.info("Specified Docker CMD as %runscript.")
    else:
        logger.info("Specified Docker ENTRYPOINT as %runscript.")

    additions = ADD(image=image,
                    auth=auth,
                    disable_cache=disable_cache)

    # Extract layers to filesystem
    for layer in additions['layers']:
        if not os.path.exists(layer):

            download_folder = os.path.dirname(layer)
            image_id,_ = os.path.splitext(os.path.basename(layer))

            targz = get_layer(image_id=image_id,
                              namespace=additions['image']['namespace'],
                              repo_name=additions['image']['repo_name'],
                              registry=additions['image']['registry'],
                              download_folder=download_folder,
                              auth=auth)

        # Extract image and remove tar
        output = extract_tar(targz,rootfs)
        if output is None:
            logger.error("Error extracting image: %s", targz)
            sys.exit(1)
        if disable_cache == True:
            os.remove(targz)
               
    # Generate runscript
    runscript = create_runscript(manifest=manifest,
                                 base_dir=rootfs,
                                 includecmd=includecmd)

    #TODO: NOT WRITTEN YET. I'm tired. :)
    if metadata_dir != None:

        # Generate environment file for Docker
        extract_env(metadata_dir=metadata_dir,
                    includecmd=includecmd,
                    manifest=manifest)

        # Generate labels files
        extract_labels(metadata_dir=metadata_dir,
                       includecmd=includecmd,
                       manifest=manifest)

    # When we finish, clean up images
    if disable_cache == True:
        shutil.rmtree(cache_base)
    logger.info("*** FINISHING DOCKER IMPORT PYTHON PORTION ****\n")



def ADD(image,metadata_dir=None,auth=None):
    '''run is the main script that will obtain docker layers, runscript information (either entrypoint
    or cmd), and environment, and either return the list of files to extract (in case of add 
    :param image: the docker image to add
    :param auth: if needed, an authentication header (default None)
    :param metadata_dir: the folder to write layers file to. If not defined, not written.
    :param includecmd: use CMD as %runscript instead of ENTRYPOINT (default False)
    :returns additions: a dict with "layers" and "manifest" for further parsing
    '''

    logger.debug("Starting Docker ADD, only includes file and folder objects")
    logger.info("Docker image: %s", image)

    # Input Parsing ----------------------------
    # Parse image name, repo name, and namespace
    image = parse_image_uri(image=image,uri="docker://")
    logger.info("Docker image path: %s/%s:%s", namespace,repo_name,repo_tag)

    # IMAGE METADATA -------------------------------------------
    # Use Docker Registry API (version 2.0) to get images ids, manifest

    # Get an image manifest - has image ids to parse, and will be
    # used later to get Cmd
    manifest = get_manifest(repo_name=image['repo_name'],
                            namespace=image['namespace'],
                            repo_tag=image['repo_tag'],
                            registry=image['registry'],
                            auth=auth)

    # Get images from manifest using version 2.0 of Docker Registry API
    images = get_images(manifest=manifest)

    #  DOWNLOAD LAYERS -------------------------------------------
    # Each is a .tar.gz file, obtained from registry with curl
       
    # Get the cache (or temporary one) for docker
    cache_base = get_cache(subfolder="docker", 
                           disable_cache=disable_cache)

    layers = []
    for image_id in images:
        targz = "%s/%s.tar.gz" %(cache_base,image_id)
        layers.append(targz) # in case we want a list at the end

    # If the user wants us to write the layers to file, do it.
    if metadata_dir != None:

        # Standard for layerfile is under SINGULARITY_METADATA_FOLDER/.layers
        metadata_file = "%s/%s" %(metadata_dir, LAYERFILE)

        # Question - here we will have /tmp paths - should this be changed
        # after they are downloaded, kept ok as is, or the file removed?
        logger.debug("Writing Docker layers files to %s", metadata_file)

        #TODO: need to have a lock of some kind here, or tmp.
        write_file(metadata_file,"\n".join(layers))

    # Return additions dictionary
    additions = { "layers": layers,
                  "image" : image,
                  "manifest": manifest }

    return additions
