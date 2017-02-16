'''

api.py: Singularity Hub helper functions for Singularity in Python

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

from shell import parse_image_uri

from shub.api import (
    download_image, 
    get_manifest
    get_image_name
)

from utils import (
    add_http,
    api_get, 
    get_cache,
    write_file
)

from logman import logger
import json
import re
import os
import tempfile
try:
    from urllib.error import HTTPError
except ImportError:
    from urllib2 import HTTPError


def PULL(image,rootfs=None,pull_folder=None):
    '''PULL will retrieve a Singularity Hub image and download to the local file
    system. If a rootfs is provided, the path to the image is written to a file
    called SINGULARITY_RUNDIR and SINGULARITY_IMAGE here, with only the purpose
    of passing the variable up.
    '''
    manifest = get_manifest(image)
    if pull_folder == None:
        cache_base = get_cache(subfolder="shub", 
                                   disable_cache = args.disable_cache)
    else:
        cache_base = pull_folder

    # The image name is the md5 hash, download if it's not there
    image_name = get_image_name(manifest)
    image_file = "%s/%s" %(cache_base,image_name)
    if not os.path.exists(image_file):
        image_file = download_image(manifest=manifest,
                                    download_folder=cache_base)
    else:
        print("Image already exists at %s, skipping download." %image_file)
    logger.info("Singularity Hub Image Download: %s", image_file)
       
    # If singularity_rootfs is provided, write metadata to it
    if rootfs != None:
        logger.debug("Writing SINGULARITY_RUNDIR and SINGULARITY_IMAGE to %s",rootfs)
        write_file("%s/SINGULARITY_RUNDIR" %rootfs, os.path.dirname(image_file))
        write_file("%s/SINGULARITY_IMAGE" %rootfs, image_file)
