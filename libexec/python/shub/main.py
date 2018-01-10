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
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))) # noqa

from shell import parse_image_uri

from .api import (
    extract_metadata,
    get_image_name,
    SingularityApiConnection
)

from sutils import (
    get_cache,
    write_file
)

from defaults import SHUB_PREFIX

from message import bot
import json
import re
import os


def SIZE(image, contentfile=None):
    '''size is intended to be run before an import,
    to return to the contentfile a list of sizes
    (one per layer) corresponding with the layers
    that will be downloaded for image
    '''
    bot.debug("Starting SHub SIZE, will get size from manifest")
    bot.debug("Singularity Hub image: %s" % image)
    client = SingularityApiConnection(image=image)
    manifest = client.get_manifest()
    if 'size_mb' in manifest:  # sregistry
        size = manifest['size_mb']
    else:
        size = manifest['metrics']['size']
    if contentfile is not None:
        write_file(contentfile, str(size), mode="w")
    return size


def PULL(image, download_folder=None, layerfile=None):
    '''PULL will retrieve a Singularity Hub image and
    download to the local file system, to the variable
    specified by SINGULARITY_PULLFOLDER.
    :param image: the singularity hub image name
    :param download folder: the folder to pull the image to.
    :param layerfile: if defined, write pulled image to file
    '''
    client = SingularityApiConnection(image=image)
    manifest = client.get_manifest()

    if download_folder is None:
        cache_base = get_cache(subfolder="shub")
    else:
        cache_base = os.path.abspath(download_folder)

    bot.debug("Pull folder set to %s" % cache_base)

    # The image name is the md5 hash, download if it's not there
    image_name = get_image_name(manifest)

    # Did the user specify an absolute path?
    custom_folder = os.path.dirname(image_name)
    if custom_folder not in [None, ""]:
        cache_base = custom_folder
        image_name = os.path.basename(image_name)

    image_file = "%s/%s" % (cache_base, image_name)

    bot.debug('Pulling to %s' % image_file)
    if not os.path.exists(image_file):
        image_file = client.download_image(manifest=manifest,
                                           download_folder=cache_base)
    else:
        if not bot.is_quiet():  # not --quiet
            print("Image already exists at %s, skipping download" % image_file)

    if not bot.is_quiet():  # not --quiet
        print("Singularity Hub Image Download: %s" % image_file)

    manifest = {'image_file': image_file,
                'manifest': manifest,
                'cache_base': cache_base,
                'image': image}

    if layerfile is not None:
        bot.debug("Writing Singularity Hub image path to %s" % layerfile)
        write_file(layerfile, image_file, mode="w")

    return manifest


def IMPORT(image, layerfile, labelfile=None):
    '''IMPORT takes one more step than ADD, returning
    the image written to a layerfile plus metadata written
    to the metadata base in rootfs.
    :param image: the singularity hub image name
    '''
    manifest = PULL(image, layerfile=layerfile)

    # Write metadata to base
    manifest['metadata'] = extract_metadata(manifest=manifest['manifest'],
                                            labelfile=labelfile,
                                            prefix="%s_" % SHUB_PREFIX)
    return manifest
