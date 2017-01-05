'''
shell.py: Docker shell parsing functions for Singularity in Python
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
sys.path.append('..') # parent directory

from logman import logger
from docker.api import get_tags
import json
import re
import os


def parse_image_uri(image,uri,default_namespace=None):
    '''parse_image_uri will return a json structure with a repo name, tag, and
    namespace.
    :param image: the string provided on command line for the image name, eg: ubuntu:latest
    :param uri: the uri (eg, docker:// to remove)
    :default_namespace: if not provided, will use "library"
    :returns parsed: a json structure with repo_name, repo_tag, and namespace
    '''
    if default_namespace == None:
        default_namespace = "library"

    # First split the docker image name by /
    image = image.replace(uri,'')
    image = image.split('/')

    # If there are two parts, we have namespace with repo (and maybe tab)
    if len(image) >= 2:
        namespace = image[0]
        image = image[1]

    # Otherwise, we must be using library namespace
    else:
        namespace = default_namespace
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

    logger.info("Repo Name: %s", repo_name)
    logger.info("Repo Tag: %s", repo_tag)
    logger.info("Namespace: %s", namespace)

    parsed = {'repo_name':repo_name,
              'repo_tag':repo_tag,
              'namespace':namespace }
    return parsed


def get_tags_shell(image,uri,default_namespace=None):
    '''get_tags_shell is a wrapper to run docker.api.get_tags with additional parsing
    of the input string. It is assumed that a tag is not provided.
    :image: the image name to be parsed by parse_image_uri
    '''
    parsed = parse_image_uri(image,uri,default_namespace=None)
    repo_name = parsed['repo_name']
    namespace = parsed['namespace']

    return get_tags(namespace=namespace,
                    repo_name=repo_name)
