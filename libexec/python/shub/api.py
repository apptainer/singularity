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
import os
sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__), os.path.pardir)))
sys.path.append('..') # parent directory

from shell import (
    remove_image_uri
)

from sutils import (
    add_http,
    is_number,
    read_file
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
except:
    from urllib.parse import unquote

# Shub API Class  ----------------------------------------------------------------------------

class SingularityApiConnection(ApiConnection):

    def __init__(self,**kwargs):
        self.token = None
        self.api_base = SHUB_API_BASE
        if 'image' in kwargs:
            self.load_image(kwargs['image'])
        if 'token' in kwargs:
            self.token = kwargs['token']
        super(SingularityApiConnection, self).__init__(**kwargs)


    def load_image(self,image):
        self.image = image
        if not is_number(image):
            self.image = remove_image_uri(image,quiet=True)
          

    def _get_token(self,domain=None,token_folder=None):
        '''_get_token is not currently in use, as API is open to all. When needed,
        can subclass get_token function.
        authenticate will authenticate the user with Singularity Hub. This means
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
                    domain = SHUB_API_BASE
                print('''Please obtain token from %s/token
                         and save to .shub in your $HOME folder''' %(domain))
                sys.exit(1)
        return token


    def get_manifest(self,image=None,registry=None):
        '''get_image will return a json object with image metadata, based on a unique id.
        :param image: the image name, either an id, or a repo name, tag, etc.
        :param registry: the registry (hub) to use, if not defined, default is used
        '''
        if image == None:
            image = self.image
        if registry == None:
            registry = self.api_base
        registry = add_http(registry) # make sure we have a complete url

        # Numeric images have slightly different endpoint from named
        if is_number(image) == True:
            base = "%s/containers/%s" %(registry,image)
        else:
            base = "%s/container/%s" %(registry,image)

        # ---------------------------------------------------------------
        # If we eventually have private images, need to authenticate here       
        # --------------------------------------------------------------- 

        response = self.get(base)
        try:
            response = json.loads(response)
        except:
            print("Error getting image manifest using url %s" %(base))
            sys.exit(1)
        return response


    def download_image(self,manifest,download_folder=None,extract=True):
        '''download_image will download a singularity image from singularity
        hub to a download_folder, named based on the image version (commit id)
        :param manifest: the manifest obtained with get_manifest
        :param download_folder: the folder to download to, if None, will be pwd
        :param extract: if True, will extract image to .img and return that.
        '''
        image_file = get_image_name(manifest)

        if not bot.is_quiet():
            print("Found image %s:%s" %(manifest['name'],manifest['branch']))
            print("Downloading image... %s" %(image_file))

        if download_folder is not None:
            image_file = "%s/%s" %(download_folder,image_file)
        url = manifest['image']

        # Download image file atomically, streaming
        image_file = self.download_atomically(url=url,
                                              file_name=image_file)

        if extract == True:
            if not bot.is_quiet():
                print("Decompressing %s" %image_file)
            os.system('gzip -d -f %s' %(image_file))
            image_file = image_file.replace('.gz','')
        return image_file



# Various Helpers ---------------------------------------------------------------------------------
def get_image_name(manifest,extension='img.gz'):
    '''get_image_name will return the image name for a manifest
    :param manifest: the image manifest with 'image' as key with download link
    :param use_hash: use the image hash instead of name
    '''
    from defaults import SHUB_CONTAINERNAME, SHUB_NAMEBYCOMMIT, SHUB_NAMEBYHASH
   
    # First preference goes to a custom name
    if SHUB_CONTAINERNAME is not None:
        for replace in [" ",".gz",".img"]:
            SHUB_CONTAINERNAME = SHUB_CONTAINERNAME.replace(replace,"")
        image_name = "%s.%s" %(SHUB_CONTAINERNAME,extension)

    # Second preference goes to commit
    elif SHUB_NAMEBYCOMMIT is not None:
        image_name = "%s.%s" %(manifest['version'],extension)

    elif SHUB_NAMEBYHASH is not None:
        image_url = os.path.basename(unquote(manifest['image']))
        image_name = re.findall(".+[.]%s" %(extension),image_url)[0]
    
    # Default uses the image name-branch
    else:
        image_name = "%s-%s.%s" %(manifest['name'].replace('/','-'),
                                  manifest['branch'].replace('/','-'),
                                  extension)          
    if not bot.is_quiet():
        print("Singularity Hub Image: %s" %image_name)
    return image_name



def extract_metadata(manifest,labelfile=None,prefix=None):
    '''extract_metadata will write a file of metadata from shub
    :param manifest: the manifest to use
    '''
    if prefix is None:
        prefix = ""
    prefix = prefix.upper()

    metadata = manifest.copy()
    remove_fields = ['files','spec','metrics']
    for remove_field in remove_fields:
        if remove_field in metadata:
            del metadata[remove_field]

    if labelfile is not None:
        for key,value in metadata.items():
            key = "%s%s" %(prefix,key)
            value = ADD(key=key,
                        value=value,
                        jsonfile=labelfile,
                        force=True)

        bot.verbose("Saving Singularity Hub metadata to %s" %labelfile)    
    return metadata
