'''
shell.py: General shell parsing functions for Singularity in Python

Copyright (c) 2017, Vanessa Sochat. All rights reserved. 
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

from logman import logger

from defaults import (
    API_BASE as default_registry,
    NAMESPACE as default_namespace,
    TAG as default_tag
)

from utils import is_number
import json
import re
import os


def get_image_uri(image):
    '''get_image_uri will parse a uri sent from Singularity to determine if it's 
    singularity (shub://) or docker (docker://)
    :param image: the complete image uri (example: docker://ubuntu:latest
    '''
    image_uri = None
    image = image.replace(' ','')
    match = re.findall('^[A-Za-z0-9-]+[:]//',image)

    if len(match) == 0:
        logger.warning("Could not detect any uri in %s",image)
    else:
        image_uri = match[0].lower()
        logger.debug("Found uri %s",image_uri)
    return image_uri


def remove_image_uri(image,image_uri=None):
    '''remove_image_uri will return just the image name
    '''
    if image_uri == None:
        image_uri = get_image_uri(image)

    image = image.replace(' ','')
        
    if image_uri != None:
        image = image.replace(image_uri,'')
    return image


def parse_image_uri(image,uri=None):
    '''parse_image_uri will return a json structure with a registry, 
    repo name, tag, and namespace, intended for Docker.
    :param image: the string provided on command line for the image name, eg: ubuntu:latest
    :param uri: the uri (eg, docker:// to remove), default uses ""
    ::note uri is maintained as a variable so we have some control over allowed
    :returns parsed: a json structure with repo_name, repo_tag, and namespace
    '''

    if uri == None:
        uri = ""

    # Be absolutely sure there are not comments
    image = image.split('#')[0]

    # Get rid of any uri, and split the tag
    image = image.replace(uri,'')
    image = image.split(':')

    # If there are two parts, we have a tag
    if len(image) == 2:
        repo_tag = image[1]
        image = image[0]

    else:
        image = image[0]
        repo_tag = default_tag

    # Now look for registry, namespace, repo
    image = image.split('/')

    if len(image) == 3:
        registry = image[0]
        namespace = image[1]
        repo_name = image[2]

    elif len(image) == 2:
        registry = default_registry
        namespace = image[0]
        repo_name = image[1]

    else:
        registry = default_registry
        namespace = default_namespace
        repo_name = image[0]

    logger.info("Registry: %s", registry)
    logger.info("Namespace: %s", namespace)
    logger.info("Repo Name: %s", repo_name)
    logger.info("Repo Tag: %s", repo_tag)

    parsed = {'registry':registry,
              'namespace':namespace, 
              'repo_name':repo_name,
              'repo_tag':repo_tag }

    # No field should be empty
    for fieldname,value in parsed.items():
        if len(value) == 0:
            logger.error("%s found empty, check uri! Exiting.", value)
            sys.exit(1)

    return parsed
