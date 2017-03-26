'''

api.py: Docker helper functions for Singularity in Python

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
sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__), os.path.pardir)))
sys.path.append('..') # parent directory

from base import ApiConnection

from utils import (
    add_http,
    get_cache,
    create_tar,
    write_file, 
    write_singularity_infos
)

from helpers.json.main import ADD
from shell import parse_image_uri

from defaults import (
    API_BASE,
    API_VERSION,
    DOCKER_NUMBER,
    DOCKER_PREFIX,
    ENV_BASE,
    LABELFILE,
    METADATA_BASE,
    METADATA_FOLDER_NAME,
    RUNSCRIPT_COMMAND_ASIS
)

from logman import logger
import json
import re
import tempfile
try:
    from urllib.error import HTTPError
except ImportError:
    from urllib2 import HTTPError


# Docker API Class  ----------------------------------------------------------------------------

class DockerApiConnection(ApiConnection):

    def __init__(self,**kwargs):
        self.auth = None
        self.token = None
        self.api_base = API_BASE
        self.api_version = API_VERSION
        self.manifest = None
        if 'image' in kwargs:
            self.load_image(kwargs['image'])
        if 'auth' in kwargs:
            self.auth = kwargs['auth']
        if 'token' in kwargs:
            self.token = kwargs['token']
        super(DockerApiConnection, self).__init__(**kwargs)
        self.update_token()
        

    def assemble_uri(self):
        '''re-assemble the image uri, for components defined.
        '''
        image_uri = "%s-%s-%s:%s" %(self.registry,self.namespace,self.repo_name,self.repo_tag)
        if self.version is not None:
            image_uri = "%s@%s" %(image_uri,self.version)
        return image_uri


    def _init_headers(self):
        # specify wanting version 2 schema, meaning the correct order of digests returned (base to child)
        return {"Accept":'application/vnd.docker.distribution.manifest.v2+json,application/vnd.docker.distribution.manifest.list.v2+json',
               'Content-Type':'application/json; charset=utf-8'}


    def load_image(self,image):
        '''load_image parses the image uri, and loads the different image parameters into
        the client. The image should be a docker uri (eg docker://) or name of docker image.
        '''
        image = parse_image_uri(image=image,uri="docker://")
        self.repo_name = image['repo_name']
        self.repo_tag = image['repo_tag']
        self.namespace = image['namespace']
        self.version = image['version']
        self.registry = image['registry']


    def update_token(self,response=None,auth=None):
        '''update_token uses HTTP basic authentication to get a token for 
        Docker registry API V2 operations. We get here if a 401 is
        returned for a request. https://docs.docker.com/registry/spec/auth/token/
        '''

        if response == None:
            response = self.get_tags(return_response=True)

        if not isinstance(response, HTTPError):
            return None

        if response.code != 401 or "Www-Authenticate" not in response.headers:
            logger.error("Authentication error, exiting.")
            sys.exit(1)

        challenge = response.headers["Www-Authenticate"]
        match = re.match('^Bearer\s+realm="(.+)",service="(.+)",scope="(.+)",?', challenge)
        if not match:
            logger.error("Unrecognized authentication challenge, exiting.")
            sys.exit(1)

        realm = match.group(1)
        service = match.group(2)
        scope = match.group(3)

        base = "%s?service=%s&scope=%s" %(realm,service,scope)
        headers = dict()
        if auth is not None:
            headers.update(auth)

        response = self.get(base,default_headers=False,headers=headers)
        try:
            token = json.loads(response)["token"]
            token = {"Authorization": "Bearer %s" %(token) }
            self.token = token
            self.update_headers(token)
        except:
            logger.error("Error getting token for repository %s/%s, exiting.", self.namespace,self.repo_name)
            sys.exit(1)



    def get_images(self):
        '''get_images is a wrapper for get_manifest, but it additionally parses the repo_name and tag's
        images and returns the complete ids
        :param repo_name: the name of the repo, eg "ubuntu"
        :param namespace: the namespace for the image, default is "library"
        :param repo_tag: the repo tag, default is "latest"
        :param registry: the docker registry url, default will use index.docker.io
        '''

        # Get full image manifest, using version 2.0 of Docker Registry API
        if self.manifest is None:
            if self.repo_name is not None and self.namespace is not None:
                self.manifest = self.get_manifest()

            else:
                logger.error("No namespace and repo name OR manifest provided, exiting.")
                sys.exit(1)

        digests = read_digests(self.manifest)
        return digests


    def get_tags(self,return_response=False):
        '''get_tags will return the tags for a repo using the Docker Version 2.0 Registry API
        :param namespace: the namespace (eg, "library")
        :param repo_name: the name for the repo (eg, "ubuntu")
        :param registry: the docker registry to use (default will use index.docker.io)
        :param auth: authorization header (default None)
        '''
        registry = self.registry
        if registry == None:
            registry = self.api_base
        
        registry = add_http(registry) # make sure we have a complete url

        base = "%s/%s/%s/%s/tags/list" %(registry,self.api_version,self.namespace,self.repo_name)
        logger.info("Obtaining tags: %s", base)

        # We use get_tags for a testing endpoint in update_token
        response = self.get(base)
        if return_response:
            return response

        try:
            response = json.loads(response)
            return response['tags']
        except:
            logger.error("Error obtaining tags: %s", base)
            sys.exit(1)


    def get_manifest(self,old_version=False):
        '''get_manifest should return an image manifest for a particular repo and tag. 
        The image details are extracted when the client is generated.
        :param old_version: return version 1 (for cmd/entrypoint), default False
        '''
        registry = self.registry
        if registry == None:
            registry = self.api_base
        registry = add_http(registry) # make sure we have a complete url

        base = "%s/%s/%s/%s/manifests" %(registry,self.api_version,self.namespace,self.repo_name)
        if self.version is not None:
            base = "%s/%s" %(base,self.version)
        else:
            base = "%s/%s" %(base,self.repo_tag)
        logger.info("Obtaining manifest: %s", base)
    
        headers = self.headers
        if old_version == True:
            headers['Accept'] = 'application/json' 

        response = self.get(base,headers=self.headers)
        try:
            response = json.loads(response)
        except:
            # If the call fails, give the user a list of acceptable tags
            tags = self.get_tags()
            print("\n".join(tags))
            repo_uri = "%s/%s:%s" %(self.namespace,self.repo_name,self.repo_tag)
            logger.error("Error getting manifest for %s, exiting.", repo_uri)
            sys.exit(1)

        return response


    def get_layer(self,image_id,download_folder=None):
        '''get_layer will download an image layer (.tar.gz) to a specified download folder.
        :param download_folder: if specified, download to folder. Otherwise return response with raw data (not recommended)
        '''
        registry = self.registry
        if registry == None:
            registry = self.api_base
        registry = add_http(registry) # make sure we have a complete url

        # The <name> variable is the namespace/repo_name
        base = "%s/%s/%s/%s/blobs/%s" %(registry,self.api_version,self.namespace,self.repo_name,image_id)
        logger.info("Downloading layers from %s", base)
    
        if download_folder is not None:
            download_folder = "%s/%s.tar.gz" %(download_folder,image_id)

            # Update user what we are doing
            print("Downloading layer %s" %image_id)

        # Download the layer atomically
        finished_download = self.download_atomically(url=base,
                                                     file_name=download_folder)
 
        return finished_download


    def get_config(self,spec="Entrypoint",delim=None):
        '''get_config returns a particular spec (default is Entrypoint) 
        from a VERSION 1 manifest obtained with get_manifest.
        :param manifest: the manifest obtained from get_manifest
        :param spec: the key of the spec to return, default is "Entrypoint"
        :param delim: Given a list, the delim to use to join the entries. Default is newline
        '''
        
        manifest = self.get_manifest(old_version=True)

        cmd = None
        if "history" in manifest:
            for entry in manifest['history']:
                if 'v1Compatibility' in entry:
                    entry = json.loads(entry['v1Compatibility'])
                    if "config" in entry:
                        if spec in entry["config"]:
                            cmd = entry["config"][spec]

        # Standard is to include commands like ['/bin/sh']
        if isinstance(cmd,list):
            if delim is None:
                delim = "\n"
            cmd = delim.join(cmd)
        logger.info("Found Docker command (%s) %s",spec,cmd)

        return cmd


# Authentication not required ---------------------------------------------------------------------------------

def read_digests(manifest):
    '''read_layers will return a list of layers from a manifest. The function is
    intended to work with both version 1 and 2 of the schema
    :param manifest: the manifest to read_layers from
    '''

    digests = []

    # https://github.com/docker/distribution/blob/master/docs/spec/manifest-v2-2.md#image-manifest
    if 'layers' in manifest:
        layer_key = 'layers'
        digest_key = 'digest'
        logger.info('Image manifest version 2.2 found.')

    # https://github.com/docker/distribution/blob/master/docs/spec/manifest-v2-1.md#example-manifest
    elif 'fsLayers' in manifest:
        layer_key = 'fsLayers'
        digest_key = 'blobSum'
        logger.info('Image manifest version 2.1 found.')

    else:
        logger.error('Improperly formed manifest, layers or fsLayers must be present')
        sys.exit(1)

    for layer in manifest[layer_key]:
        if digest_key in layer:
            if layer[digest_key] not in digests:
                logger.info("Adding digest %s",layer[digest_key])
                digests.append(layer[digest_key])
    return digests
    


def create_runscript(manifest,includecmd=False):
    '''create_runscript will write a bash script with default "ENTRYPOINT" 
    into the base_dir. If includecmd is True, CMD is used instead. For both.
    if the result is found empty, the other is tried, and then a default used.
    :param manifest: the manifest to use to get the runscript
    :param includecmd: overwrite default command (ENTRYPOINT) default is False
    '''
    if METADATA_BASE == None:
        logger.warning('''METADATA_BASE/SINGULARITY_ROOTFS not defined in environment!
                       Will not write runscript to file, but return to function call.''')
        runscript = None
    else:
        runscript = "%s/runscript" %(METADATA_BASE)
    cmd = None

    # Does the user want to use the CMD instead of ENTRYPOINT?
    commands = ["Entrypoint","Cmd"]
    if includecmd == True:
        commands.reverse()
    configs = get_configs(manifest,commands,delim=" ")
    
    # Look for non "None" command
    for command in commands:
        if configs[command] != None:
            cmd = configs[command]
            break

    if cmd != None:
        logger.debug("Adding Docker %s as Singularity runscript..." %(command.upper()))
        logger.debug(cmd)

        # If the command is a list, join. (eg ['/usr/bin/python','hello.py']
        if isinstance(cmd,list):
            cmd = " ".join(cmd)

        if not RUNSCRIPT_COMMAND_ASIS:
            cmd = 'exec %s "$@"' %(cmd)
        cmd = "#!/bin/sh\n\n%s" %(cmd)
        logger.info("Generating runscript at %s",runscript)
        if runscript != None:
            output_file = write_file(runscript,cmd)
            return output_file
        return runscript
    print("No Docker CMD or ENTRYPOINT found, skipping runscript generation.")
    return cmd



def extract_metadata_tar(manifest,image_name,include_env=True,include_labels=True):
    '''extract_metadata_tar will write a tarfile with the environment 
    '''
    tar_file = None
    if include_env or include_labels:
        cache_base = get_cache(subfolder="docker",quiet=True)
        output_file = "%s/metadata-%s.tar.gz" %(cache_base,
                                                image_name)

        if not os.path.exists(output_file):
            files = []

            if include_env:               
                environ = extract_env(manifest)
                if environ not in [None,""]:
                    logger.debug('Adding %s to files for metadata tar',environ)
                    files.append ({'name':'./%s/env/%s-%s.sh' %(METADATA_FOLDER_NAME,
                                                                DOCKER_NUMBER,
                                                                DOCKER_PREFIX),
                                   'permission': 493, #755,'0o755'
                                   'content': environ })
            if include_labels:
                labels = extract_labels(manifest)
                if labels is not None:
                    if isinstance(labels,dict):
                        labels = json.dumps(labels)
                    logger.debug('Adding %s labels for metadata tar',labels)
                    files.append ({'name': "./%s/labels.json" %METADATA_FOLDER_NAME,
                                   'permission': 493,
                                   'content': labels })
 
            if len(files) > 0:
                tar_file = create_tar(files,output_file)

        else:
            logger.warning("metadata file %s already exists, not over-writing." %(output_file))

    return tar_file


def extract_env(manifest):
    '''extract the environment from the manifest, or return None. Used by
    functions env_extract_image, and env_extract_tar
    '''
    environ = get_config(manifest,'Env')
    if environ is not None:
        if isinstance(environ,list):
            environ = "\n".join(environ)
        environ = ["export %s" %x for x in environ.split('\n')]
        environ = "\n".join(environ)
        logger.debug("Found Docker container environment!")    
    return environ


def env_extract_image(manifest):
    '''env_extract_image will write a file of key value pairs of the environment
    to export. The manner to export must be determined by the calling process
    depending on the OS type.
    :param manifest: the manifest to use
    '''
    environ = extract_env(manifest)
    if environ is not None:
        environ_file = write_singularity_infos(base_dir=ENV_BASE,
                                               prefix=DOCKER_PREFIX,
                                               start_number=DOCKER_NUMBER,
                                               content=environ,
                                               extension='sh')
    return environ



def extract_labels(manifest,labelfile=None,prefix=None):
    '''extract_labels will write a file of key value pairs including
    maintainer, and labels.
    :param manifest: the manifest to use
    :param labelfile: if defined, write to labelfile (json)
    :param prefix: an optional prefix to add to the names
    '''
    if prefix is None:
        prefix = ""

    labels = get_config(manifest,'Labels')
    if labels is not None and len(labels) is not 0:
        logger.debug("Found Docker container labels!")    
        if labelfile is not None:
            for key,value in labels.items():
                key = "%s%s" %(prefix,key)
                value = ADD(key,value,labelfile)
    return labels


def get_config(manifest,spec="Entrypoint",delim=None):
    '''get_config returns a particular spec (default is Entrypoint) 
    from a VERSION 1 manifest obtained with get_manifest.
    :param manifest: the manifest obtained from get_manifest
    :param spec: the key of the spec to return, default is "Entrypoint"
    :param delim: Given a list, the delim to use to join the entries. Default is newline
    '''
    cmd = None
    if "history" in manifest:
        for entry in manifest['history']:
            if 'v1Compatibility' in entry:
                entry = json.loads(entry['v1Compatibility'])
                if "config" in entry:
                    if spec in entry["config"]:
                        cmd = entry["config"][spec]

    # Standard is to include commands like ['/bin/sh']
    if isinstance(cmd,list):
        if delim is None:
            delim = "\n"
        cmd = delim.join(cmd)
    logger.info("Found Docker command (%s) %s",spec,cmd)

    return cmd


def get_configs(manifest,keys,delim=None):
    '''get_configs is a wrapper for get_config to return a dictionary
    with multiple config items.
    :param manifest: the complete manifest
    :param keys: the key to find
    :param delim: given a list, combine based on this delim
    '''
    configs = dict()
    if not isinstance(keys,list):
        keys = [keys]
    for key in keys:
        configs[key] = get_config(manifest,key,delim=delim)
    return configs
