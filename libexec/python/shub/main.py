'''

main.py: Singularity Hub helper functions for Singularity in Python

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
sys.path.append('..') # parent directory

from shell import parse_image_uri

from .api import (
    download_image, 
    extract_metadata,
    get_manifest,
    get_image_name
)

from utils import (
    add_http,
    api_get, 
    get_cache,
    write_file
)

from defaults import SHUB_PREFIX

from logman import logger
import json
import re
import os
import tempfile


def PULL(image,download_folder=None,layerfile=None):
    '''PULL will retrieve a Singularity Hub image and download to the local file
    system, to the variable specified by SINGULARITY_PULLFOLDER.
    :param image: the singularity hub image name
    :param download folder: the folder to pull the image to.
    :param layerfile: if defined, write pulled image to file
    '''
    
    manifest = get_manifest(image)
    if download_folder == None:
        cache_base = get_cache(subfolder="shub")
    else:
        cache_base = download_folder

    # The image name is the md5 hash, download if it's not there
    image_name = get_image_name(manifest)
    image_file = "%s/%s" %(cache_base,image_name)
    if not os.path.exists(image_file):
        image_file = download_image(manifest=manifest,
                                    download_folder=cache_base)
    else:
        print("Image already exists at %s, skipping download." %image_file)
    logger.info("Singularity Hub Image Download: %s", image_file)

    manifest = {'image_file': image_file,
                'manifest': manifest,
                'cache_base': cache_base,
                'image': image }

    if layerfile != None:
        logger.debug("Writing Singularity Hub image path to %s", layerfile)
        write_file(layerfile,image_file,mode="w")

    return manifest



def IMPORT(image,layerfile,labelfile=None):
    '''IMPORT takes one more step than ADD, returning the image written to a layerfile
    plus metadata written to the metadata base in rootfs.
    :param image: the singularity hub image name
    '''
    manifest = PULL(image,layerfile=layerfile)

    # Write metadata to base    
    manifest['metadata'] = extract_metadata(manifest=manifest['manifest'],
                                            labelfile=labelfile,
                                            prefix="%s_" %SHUB_PREFIX)
    return manifest
