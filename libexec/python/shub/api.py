#!/usr/bin/env python

'''

api.py: Singularity Hub helper functions for python

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

import sys
sys.path.append('..') # parent directory

from utils import (
    add_http,
    api_get, 
    is_number,
    read_file,
    write_file
)
from logman import logger
import json
import os
import re

try:
    from urllib import unquote 
except:
    from urllib.parse import unquote

api_base = "singularity-hub.org/api"


def authenticate(domain=None,token_folder=None):
    '''authenticate will authenticate the user with Singularity Hub. This means
    either obtaining the token from the environment, and then trying to obtain
    the token file and reading it, and then finally telling the user to get it.
    :param domain: the domain to direct the user to for the token, default is api_base
    :param token_folder: the location of the token file, default is $HOME (~/)
    '''
    # Attempt 1: Get token from environmental variable
    token = os.environ.get("SINGULARITY_TOKEN",None)

    if token == None:
        # Is the user specifying a custom home folder?
        if token_folder == None:
            token_folder = os.environ["HOME"]

        token_file = "%s/.shub" %(token_folder)
        if os.path.exists(token_file):
            token = read_file(token_file)[0].strip('\n')
        else:
            if domain == None:
                domain = api_base
            print('''Please obtain token from %s/token
                     and save to .shub in your $HOME folder''' %(domain))
            sys.exit(1)
    return token


# Authentication required ---------------------------------------------------------------------------------
# Docker Registry Version 2.0 Functions - IN USE


def get_manifest(image,registry=None):
    '''get_image will return a json object with image metadata, based on a unique id.
    :param image: the image name, either an id, or a repo name, tag, etc.
    :param registry: the registry (hub) to use, if not defined, default is used
    '''
    if registry == None:
        registry = api_base
    registry = add_http(registry) # make sure we have a complete url

    # Numeric images have slightly different endpoint from named
    if is_number(image) == True:
        base = "%s/containers/%s" %(registry,image)
    else:
        base = "%s/container/%s" %(registry,image)

    # ---------------------------------------------------------------
    # If we eventually have private images, need to authenticate here       
    # --------------------------------------------------------------- 

    response = api_get(base)
    try:
        response = json.loads(response)
    except:
        print("Error getting image manifest using url %s" %(base))
        sys.exit(1)
    return response


def download_image(manifest,download_folder=None,extract=True):
    '''download_image will download a singularity image from singularity
    hub to a download_folder, named based on the image version (commit id)
    :param manifest: the manifest obtained with get_manifest
    :param download_folder: the folder to download to, if None, will be pwd
    :param extract: if True, will extract image to .img and return that.
    '''    
    image_file = get_image_name(manifest)

    print("Found image %s:%s" %(manifest['name'],manifest['branch']))
    print("Downloading image... %s" %(image_file))

    if download_folder != None:
        image_file = "%s/%s" %(download_folder,image_file)
    url = manifest['image']
    image_file = api_get(url,stream=image_file)
    if extract == True:
        print("Decompressing", image_file)
        os.system('gzip -d -f %s' %(image_file))
        image_file = image_file.replace('.gz','')
    return image_file


# Various Helpers ---------------------------------------------------------------------------------
def get_image_name(manifest,extension='img.gz',use_commit=True):
    '''get_image_name will return the image name for a manifest
    :param manifest: the image manifest with 'image' as key with download link
    :param use_commit: use the commit id to name the image (default) otherwise use md5sum
    '''
    image_url = os.path.basename(unquote(manifest['image']))
    image_name = re.findall(".+[.]%s" %(extension),image_url)
    if len(image_name) > 0:
        image_name = image_name[0]
        if use_commit == True:
            image_name = "%s.img.gz" %(manifest["version"])            
        logger.info("Singularity Hub Image: %s", image_name)
        return image_name
    else:
        logger.error("Singularity Hub Image not found with expected extension %s, exiting.",extension)
        sys.exit(1)
