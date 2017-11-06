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
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))) # noqa

from shell import (
    parse_image_uri,
    remove_image_uri,
)

from sutils import (
    add_http,
    clean_up,
    get_image_format,
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
        self.image = parse_image_uri(image=image,
                                     uri='shub://',
                                     default_registry=SHUB_API_BASE,
                                     quiet=True)
        # parse_image_uri may return an empty namespace cause that's allowed
        # with docker://, but not with shub://
        if len(self.image["namespace"]) == 0:
            bot.error("Namespace cannot be empty for shub:// url!")
            sys.exit(1)

    def get_manifest(self):
        '''get_image will return a json object with image metadata
        based on a unique id.

        Parameters
        ==========
        :param image: the image name, either an id
                      or a repo name, tag, etc.

        Returns
        =======
        manifest: a json manifest from the registry
        '''
        # make sure we have a complete url
        registry = add_http(self.image['registry'])
        base = "%s/api/container/%s/%s:%s" % (registry,
                                              self.image['namespace'],
                                              self.image['repo_name'],
                                              self.image['repo_tag'])
        # ------------------------------------------------------
        # If we need to authenticate, will do it here
        # ------------------------------------------------------

        # If the Hub returns 404, the image name is likely wrong
        response = self.get(base, return_response=True)
        if response.code == 404:
            msg = "Cannot find image."
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

    def download_image(self,
                       manifest,
                       download_folder=None,
                       extract=False):

        '''

        download_image will download a singularity image from singularity
        hub to a download_folder, named based on the image version (commit id)

        Parameters
        ==========
        :param manifest: the manifest obtained with get_manifest
        :param download_folder: the folder to download to, if None, will be pwd
        :param extract: if True, will extract image to .img and return that.

        Returns
        =======
        image_file: the full path to the downloaded image

        '''
        from defaults import SHUB_CONTAINERNAME

        # Returns just basename, no extension
        image_name = get_image_name(manifest)

        if not bot.is_quiet():
            print("Found image %s:%s" % (manifest['name'],
                                         manifest['branch']))

            print("Downloading image... %s" % image_name)

        url = manifest['image']

        if url is None:
            bot.error("%s is not ready for download" % image_name)
            bot.error("please try when build completes or specify tag.")
            sys.exit(1)

        if download_folder is not None:
            image_name = "%s/%s" % (download_folder, image_name)

        # Download image file atomically, streaming
        image_file = self.download_atomically(url=url,
                                              file_name=image_name,
                                              show_progress=True)

        # Compressed ext3 images need extraction
        image_type = get_image_format(image_file)
        extension = "simg"

        if image_type == "GZIP" or extract is True:

            extension = "img"
            if not image_file.endswith('.gz'):
                os.rename(image_file, "%s.gz" % image_file)
                image_file = "%s.gz" % image_file

            if not bot.is_quiet():
                print("Decompressing %s" % image_file)

            output = run_command(['gzip', '-d', '-f', image_file])
            image_file = image_file.replace('.gz', '')

            # Any error in extraction (return code not 0) will return None
            if output is None:
                bot.error('Error extracting image, cleaning up.')
                clean_up([image_file, "%s.gz" % image_file])

        # If the user has provided a default name, be true to it
        if SHUB_CONTAINERNAME is not None:
            folder = os.path.dirname(image_file)
            image_name = "%s/%s" % (folder, SHUB_CONTAINERNAME)
            os.rename(image_file, image_name)
            image_file = image_name

        # Otherwise rename to have extension matching image type
        else:
            image_name = "%s.%s" % (image_file, extension)
            os.rename(image_file, image_name)
            image_file = image_name

        return image_file


# Various Helpers -----------------------------------------------
def get_image_name(manifest):
    '''return the image name for a manifest. Does not return extension, and
       does not account for a custom name provided by the user.
       :param manifest: the image manifest with 'image'
                     as key with download link
    '''
    from defaults import (SHUB_CONTAINERNAME,
                          SHUB_NAMEBYCOMMIT,
                          SHUB_NAMEBYHASH)

    if SHUB_CONTAINERNAME is not None:
        return SHUB_CONTAINERNAME

    # Second preference goes to commit
    elif SHUB_NAMEBYCOMMIT is not None:
        if manifest.get("commit") is not None:
            return manifest['commit']
        elif manifest['version'] is not None:
            return manifest['version']

    elif SHUB_NAMEBYHASH is not None:
        image_url = os.path.basename(unquote(manifest['image']))
        image_name = re.findall('([-\w]+\.(?:img|simg|img.gz))', image_url)
        if len(image_name) > 0:
            return image_name[0].split('.')[0]

    return get_default_name(manifest)


def get_default_name(manifest, source="Hub"):
    ''' get manifest name returns the default name that is discovered
        via the image manifest. This is the fallback option in the case that
        the user doesn't ask to name by commit, hash, or custom name
    '''

    # Singularity Hub v2.0
    if "tag" in manifest and "branch" in manifest:
        version = "%s-%s" % (manifest["branch"], manifest["tag"])

    # Singularity Registry
    elif "tag" in manifest:
        version = manifest["tag"]

    # Singularity Hub v1.0
    else:
        version = manifest['branch']

    # Remove slashes
    version = version.replace('/', '-')

    # sregistry images store collection/name separately
    name = manifest['name']
    if 'frozen' in manifest:
        source = "Registry"
        name = '%s-%s' % (manifest['collection'], name)
    image_name = "%s-%s" % (name.replace('/', '-'), version)

    if not bot.is_quiet():
        print("Singularity %s Image: %s" % (source, image_name))
    return image_name


def extract_metadata(manifest, labelfile=None, prefix=None):
    '''extract_metadata will write a file of metadata from shub
    :param manifest: the manifest to use
    '''

    if prefix is None:
        prefix = ""
    prefix = prefix.upper()

    source = 'Hub'
    if 'frozen' in manifest:
        source = 'Registry'

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

        bot.verbose("Saving Singularity %s metadata to %s" % (source,
                                                              labelfile))
    return metadata
