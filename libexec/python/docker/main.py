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
import os
from defaults import DISABLE_CACHE
from utils import (
    extract_tar,
    get_cache, 
    write_file
)

from .api import (
    create_runscript,
    DockerApiConnection,
    extract_env,
    extract_labels,
    extract_metadata_tar,
)

from logman import logger
import json
import shutil
import re
import os
import tempfile


def IMPORT(image,rootfs,auth=None,includecmd=False,labelfile=None):
    '''run is the main script that will obtain docker layers, runscript information (either entrypoint
    or cmd), and environment, and either return the list of files to extract (in case of add 
    :param image: the docker image to add
    :param auth: if needed, an authentication header (default None)
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
                    layerfile=None)

    # Extract layers to filesystem
    for targz in additions['layers']:
 
        if not os.path.exists(targz):
            logger.error("Cannot find %s, was it removed by another process?")
            sys.exit(1)

        # Extract image and remove tar
        output = extract_tar(targz,rootfs)
        if DISABLE_CACHE == True:
            os.remove(targz)

        if output is None:
            logger.error("Error extracting image: %s", targz)
            sys.exit(1)
               
    # Generate runscript
    runscript = create_runscript(manifest=additions['manifestv1'],
                                 includecmd=includecmd)

    # Clean up?
    if DISABLE_CACHE == True:
        shutil.rmtree(additions['cache_base'])

    # Extract environment and labels
    extract_env(additions['manifest'])
    if labelfile is not None:
        extract_labels(manifest=additions['manifest'],
                       labelfile=labelfile,
                       prefix="DOCKER_")

    # When we finish, clean up images
    logger.info("*** FINISHING DOCKER IMPORT PYTHON PORTION ****\n")



def ADD(image,auth=None,layerfile=None):
    '''ADD is the main script that will obtain docker layers, runscript information (either entrypoint
    or cmd), and environment, and either return the list of files to extract (in case of add 
    :param image: the docker image to add
    :param auth: if needed, an authentication header (default None)
    :param layerfile: If True, write layers to METADATA_BASE/.layers
    :returns additions: a dict with "layers", "manifest", "cache_base" for further parsing
    '''

    logger.debug("Starting Docker ADD, only includes file and folder objects")
    logger.info("Docker image: %s", image)

    # Input Parsing ----------------------------
    # Parse image name, repo name, and namespace
    client = DockerApiConnection(image=image,auth=auth)

    docker_image_uri = "Docker image path: %s/%s/%s:%s" %(client.registry,
                                                          client.namespace,
                                                          client.repo_name,
                                                          client.repo_tag)
    if client.version is not None:
        docker_image_uri = "%s@%s" %(docker_image_uri,client.version)
    logger.info(docker_image_uri)


    # IMAGE METADATA -------------------------------------------
    # Use Docker Registry API (version 2.0) to get images ids, manifest

    # Get images from manifest using version 2.0 of Docker Registry API
    images = client.get_images()
    manifest = client.manifest

    #  DOWNLOAD LAYERS -------------------------------------------
    # Each is a .tar.gz file, obtained from registry with curl
       
    # Get the cache (or temporary one) for docker
    cache_base = get_cache(subfolder="docker")

    layers = []
    for image_id in images:

        targz = "%s/%s.tar.gz" %(cache_base,image_id)
        if not os.path.exists(targz):
            targz = client.get_layer(image_id=image_id,
                                     download_folder=cache_base)

        layers.append(targz) # in case we want a list at the end

    # Add the environment export
    manifestv1 = client.get_manifest(old_version=True)
    tar_file = extract_metadata_tar(manifestv1,client.assemble_uri())
    logger.debug('Tar file with Docker env and labels: %s' %(tar_file))

    # If the user wants us to write the layers to file, do it.
    if layerfile is not None:
        logger.debug("Writing Docker layers files to %s", layerfile)
        write_file(layerfile,"\n".join(layers),mode="w")
        if tar_file is not None:
            write_file(layerfile,"\n%s" %tar_file,mode="a")

    # We need version1 of the manifest for CMD/ENTRYPOINT
    manifestv1 = client.get_manifest(old_version=True)

    # Return additions dictionary
    additions = { "layers": layers,
                  "image" : image,
                  "manifest": manifest,
                  "manifestv1":manifestv1,
                  "cache_base":cache_base,
                  "metadata": tar_file }

    return additions
