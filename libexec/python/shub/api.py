#!/usr/bin/env python

'''

api.py: Singularity Hub helper functions for python

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
sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__),
                os.path.pardir)))  # noqa
sys.path.append('..')  # noqa

from shell import (
    remove_image_uri
)

from sutils import (
    add_http,
    clean_up,
    is_number,
    read_file,
    run_command
)

from base import ApiConnection

from helpers.json.main import ADD

from defaults import (
    SHUB_API_BASE
)

from message import bot
import json
import re

try:
    from urllib import unquote
except Exception:
    from urllib.parse import unquote


# Shub API Class  ----------------------------------------------

class SingularityApiConnection(ApiConnection):

    def __init__(self, **kwargs):
        self.token = None
        self.api_base = SHUB_API_BASE
        if 'image' in kwargs:
            self.load_image(kwargs['image'])
        if 'token' in kwargs:
            self.token = kwargs['token']
        super(SingularityApiConnection, self).__init__(**kwargs)

    def load_image(self, image):
        self.image = image
        if not is_number(image):
            self.image = remove_image_uri(image, quiet=True)

    def get_manifest(self, image=None, registry=None):
        '''get_image will return a json object with image metadata
        based on a unique id.
        :param image: the image name, either an id
                      or a repo name, tag, etc.
        :param registry: the registry (hub) to use
                         if not defined, default is used
        '''
        if image is None:
            image = self.image
        if registry is None:
            registry = self.api_base

        # make sure we have a complete url
        registry = add_http(registry)

        # Numeric images have slightly different endpoint from named
        if is_number(image) is True:
            base = "%s/containers/%s" % (registry, image)
        else:
            base = "%s/container/%s" % (registry, image)

        # ------------------------------------------------------
        # If we need to authenticate, will do it here
        # ------------------------------------------------------

        # If the Hub returns 404, the image name is likely wrong
        response = self.get(base, return_response=True)
        if response.code == 404:
            msg = "Cannot find image %s." % image
            msg += " Is your capitalization correct?"
            bot.error(msg)
            sys.exit(1)

        try:
            response = response.read().decode('utf-8')
            response = json.loads(response)

        except Exception:
            print("Error getting image manifest using url %s" % base)
            sys.exit(1)

        return response

    def download_image(self, manifest, download_folder=None, extract=True):
        '''download_image will download a singularity image from singularity
        hub to a download_folder, named based on the image version (commit id)
        :param manifest: the manifest obtained with get_manifest
        :param download_folder: the folder to download to, if None, will be pwd
        :param extract: if True, will extract image to .img and return that.
        '''
        image_file = get_image_name(manifest)

        if not bot.is_quiet():
            print("Found image %s:%s" % (manifest['name'],
                                         manifest['branch']))

            print("Downloading image... %s" % image_file)

        url = manifest['image']

        if url is None:
            bot.error("%s is not ready for download" % image_file)
            bot.error("please try when build completed or specify tag.")
            sys.exit(1)

        if download_folder is not None:
            image_file = "%s/%s" % (download_folder, image_file)

        # Download image file atomically, streaming
        image_file = self.download_atomically(url=url,
                                              file_name=image_file,
                                              show_progress=True)

        if extract is True:
            if not bot.is_quiet():
                print("Decompressing %s" % image_file)
            output = run_command(['gzip', '-d', '-f', image_file])
            image_file = image_file.replace('.gz', '')

            # Any error in extraction (return code not 0) will return None
            if output is None:
                bot.error('Error extracting image, cleaning up.')
                clean_up([image_file, "%s.gz" % image_file])

        return image_file


# Various Helpers -----------------------------------------------
def get_image_name(manifest, extension='img.gz'):
    '''return the image name for a manifest
    :param manifest: the image manifest with 'image'
                     as key with download link
    :param use_hash: use the image hash instead of name
    '''
    from defaults import (SHUB_CONTAINERNAME,
                          SHUB_NAMEBYCOMMIT,
                          SHUB_NAMEBYHASH)

    # First preference goes to a custom name
    if SHUB_CONTAINERNAME is not None:
        for replace in [" ", ".gz", ".img"]:
            SHUB_CONTAINERNAME = SHUB_CONTAINERNAME.replace(replace, "")
        image_name = "%s.%s" % (SHUB_CONTAINERNAME, extension)

    # Second preference goes to commit
    elif SHUB_NAMEBYCOMMIT is not None:
        image_name = "%s.%s" % (manifest['version'], extension)

    elif SHUB_NAMEBYHASH is not None:
        image_url = os.path.basename(unquote(manifest['image']))
        image_name = re.findall(".+[.]%s" % (extension), image_url)[0]

    # Default uses the image name-branch
    else:
        image_name = "%s-%s.%s" % (manifest['name'].replace('/', '-'),
                                   manifest['branch'].replace('/', '-'),
                                   extension)
    if not bot.is_quiet():
        print("Singularity Hub Image: %s" % image_name)
    return image_name


def extract_metadata(manifest, labelfile=None, prefix=None):
    '''extract_metadata will write a file of metadata from shub
    :param manifest: the manifest to use
    '''
    if prefix is None:
        prefix = ""
    prefix = prefix.upper()

    metadata = manifest.copy()
    remove_fields = ['files', 'spec', 'metrics']
    for remove_field in remove_fields:
        if remove_field in metadata:
            del metadata[remove_field]

    if labelfile is not None:
        for key, value in metadata.items():
            key = "%s%s" % (prefix, key)
            value = ADD(key=key,
                        value=value,
                        jsonfile=labelfile,
                        force=True)

        bot.verbose("Saving Singularity Hub metadata to %s" % labelfile)
    return metadata
