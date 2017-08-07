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

from message import bot

from defaults import (
    API_BASE as default_registry,
    NAMESPACE as default_namespace,
    TAG as default_tag
)

from sutils import is_number
import json
import re
import os


def get_image_uri(image, quiet=False):
    '''get_image_uri will parse a uri sent from Singularity
       to determine if it's  singularity (shub://)
       or docker (docker://)
    :param image: the complete image uri (example: docker://ubuntu:latest
    '''
    image_uri = None
    image = image.replace(' ', '')
    match = re.findall('^[A-Za-z0-9-]+[:]//', image)

    if len(match) == 0:
        if not quiet:
            bot.warning("Could not detect any uri in %s" % image)
    else:
        image_uri = match[0].lower()
        if not quiet:
            bot.debug("Found uri %s" % (image_uri))
    return image_uri


def remove_image_uri(image, image_uri=None, quiet=False):
    '''remove_image_uri will return just the image name
    '''
    if image_uri is None:
        image_uri = get_image_uri(image, quiet=quiet)

    image = image.replace(' ', '')

    if image_uri is not None:
        image = image.replace(image_uri, '')
    return image


def parse_image_uri(image, uri=None, quiet=False):
    '''parse_image_uri will return a json structure with a registry,
    repo name, tag, and namespace, intended for Docker.
    :param image: the string provided on command line for
                  the image name, eg: ubuntu:latest
    :param uri: the uri (eg, docker:// to remove), default uses ""
    ::note uri is maintained as variable so we have some control over allowed
    :returns parsed: a json structure with repo_name, repo_tag, and namespace
    '''

    if uri is None:
        uri = ""

    # Be absolutely sure there are not comments
    image = image.split('#')[0]

    # Get rid of any uri, and split the tag
    image = image.replace(uri, '')

    # Does the uri have a digest or Github tag (version)?
    image = image.split('@')
    version = None
    if len(image) == 2:
        version = image[1]

    image = image[0]
    image = image.split(':')

    # If there are three parts, we have port and tag
    if len(image) == 3:
        repo_tag = image[2]
        image = "%s:%s" % (image[0], image[1])

    # If there are two parts, we have port or tag
    elif len(image) == 2:
        # If there isn't a slash in second part, we have a tag
        if image[1].find("/") == -1:
            repo_tag = image[1]
            image = image[0]
        # Otherwise we have a port and we merge the path
        else:
            image = "%s:%s" % (image[0], image[1])
            repo_tag = default_tag
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

    if not quiet:
        bot.verbose("Registry: %s" % registry)
        bot.verbose("Namespace: %s" % namespace)
        bot.verbose("Repo Name: %s" % repo_name)
        bot.verbose("Repo Tag: %s" % repo_tag)
        bot.verbose("Version: %s" % version)

    parsed = {'registry': registry,
              'namespace': namespace,
              'repo_name': repo_name,
              'repo_tag': repo_tag}

    # No field should be empty
    for fieldname, value in parsed.items():
        if len(value) == 0:
            bot.error("%s found empty, check uri! Exiting." % value)
            sys.exit(1)

    # Version is not required
    parsed['version'] = version

    return parsed
