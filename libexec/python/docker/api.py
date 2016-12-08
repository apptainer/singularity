'''

api.py: Docker helper functions for Singularity in Python

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

from utils import api_get, write_file, add_http
from logman import logger
import json
import re
import os
import tempfile
try:
    from urllib.error import HTTPError
except ImportError:
    from urllib2 import HTTPError

api_base = "index.docker.io"
api_version = "v2"

# Authentication not required ---------------------------------------------------------------------------------

def create_runscript(cmd,base_dir):
    '''create_runscript will write a bash script with command "cmd" into the base_dir
    :param cmd: the command to write into the bash script
    :param base_dir: the base directory to write the runscript to
    '''
    runscript = "%s/singularity" %(base_dir)
    content = 'exec %s "$@"' %(cmd)
    logger.info("Generating runscript at %s",runscript)
    output_file = write_file(runscript,content)
    return output_file


def get_token(namespace,repo_name,registry=None,auth=None):
    '''get_token uses HTTP basic authentication to get a token for Docker registry API V2 operations
    :param namespace: the namespace for the image
    :param repo_name: the name of the repo, eg "ubuntu"
    :param registry: the docker registry to use
    :param auth: authorization header (default None)
    :: note
            # https://docs.docker.com/registry/spec/auth/token/
    '''
    if registry == None:
        registry = api_base
    registry = add_http(registry) # make sure we have a complete url

    # Check if we need a token at all by probing the tags/list endpoint.  This
    # is an arbitrary choice, ideally we should always attempt without a token
    # and then retry with a token if we received a 401.
    base = "%s/%s/%s/%s/tags/list" %(registry,api_version,namespace,repo_name)
    response = api_get(base, default_header=False)
    if not isinstance(response, HTTPError):
        # No token required for registry.
        return None

    if response.code != 401 or "WWW-Authenticate" not in response.headers:
        logger.error("Authentication error for registry %s, exiting.", registry)
        sys.exit(1)

    challenge = response.headers["WWW-Authenticate"]
    match = re.match('^Bearer\s+realm="([^"]+)",service="([^"]+)",scope="([^"]+)"\s*$', challenge)
    if not match:
        logger.error("Unrecognized authentication challenge from registry %s, exiting.", registry)
        sys.exit(1)

    realm = match.group(1)
    service = match.group(2)
    scope = match.group(3)

    base = "%s?service=%s&scope=%s" % (realm, service, scope)
    headers = dict()
    if auth is not None:
        headers.update(auth)

    response = api_get(base,default_header=False,headers=headers)
    try:
        token = json.loads(response)["token"]
        token = {"Authorization": "Bearer %s" %(token) }
        return token
    except:
        logger.error("Error getting token for repository %s/%s, exiting.", namespace,repo_name)
        sys.exit(1)



# Authentication required ---------------------------------------------------------------------------------
# Docker Registry Version 2.0 Functions - IN USE


def get_images(repo_name=None,namespace=None,manifest=None,repo_tag="latest",registry=None,auth=None):
    '''get_images is a wrapper for get_manifest, but it additionally parses the repo_name and tag's
    images and returns the complete ids
    :param repo_name: the name of the repo, eg "ubuntu"
    :param namespace: the namespace for the image, default is "library"
    :param repo_tag: the repo tag, default is "latest"
    :param registry: the docker registry url, default will use index.docker.io
    '''

    # Get full image manifest, using version 2.0 of Docker Registry API
    if manifest == None:
        if repo_name != None and namespace != None:

            # Custom header to specify we want a list of the version 2 schema, meaning the correct order of digests returned (base to child)
            headers = {"Accept":'application/vnd.docker.distribution.manifest.v2+json,application/vnd.docker.distribution.manifest.list.v2+json'}
            manifest = get_manifest(repo_name=repo_name,
                                    namespace=namespace,
                                    repo_tag=repo_tag,
                                    registry=registry,
                                    headers=headers,
                                    auth=auth)
        else:
            logger.error("No namespace and repo name OR manifest provided, exiting.")
            sys.exit(1)

    digests = read_digests(manifest)
    return digests


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
    

def get_tags(namespace,repo_name,registry=None,auth=None):
    '''get_tags will return the tags for a repo using the Docker Version 2.0 Registry API
    :param namespace: the namespace (eg, "library")
    :param repo_name: the name for the repo (eg, "ubuntu")
    :param registry: the docker registry to use (default will use index.docker.io)
    :param auth: authorization header (default None)
    '''
    if registry == None:
        registry = api_base
    registry = add_http(registry) # make sure we have a complete url

    base = "%s/%s/%s/%s/tags/list" %(registry,api_version,namespace,repo_name)
    logger.info("Obtaining tags: %s", base)

    token = get_token(registry=registry,
                      repo_name=repo_name,
                      namespace=namespace,
                      auth=auth)

    response = api_get(base,headers=token)
    try:
        response = json.loads(response)
        return response['tags']
    except:
        logger.error("Error obtaining tags: %s", base)
        sys.exit(1)


def get_manifest(repo_name,namespace,repo_tag="latest",registry=None,auth=None,headers=None):
    '''get_manifest should return an image manifest for a particular repo and tag. The token is expected to
    be from version 2.0 (function above)
    :param repo_name: the name of the repo, eg "ubuntu"
    :param namespace: the namespace for the image, default is "library"
    :param repo_tag: the repo tag, default is "latest"
    :param registry: the docker registry to use (default will use index.docker.io)
    :param auth: authorization header (default None)
    :param headers: dictionary of custom headers to add to token header (to get more specific manifest)
    '''
    if registry == None:
        registry = api_base
    registry = add_http(registry) # make sure we have a complete url

    base = "%s/%s/%s/%s/manifests/%s" %(registry,api_version,namespace,repo_name,repo_tag)
    logger.info("Obtaining manifest: %s", base)
    
    # Format the token, and prepare a header
    token = get_token(registry=registry,
                      repo_name=repo_name,
                      namespace=namespace,
                      auth=auth)

    # Add ['Accept'] header to specify version 2 of manifest
    if headers != None:
        if token != None:
            token.update(headers)
        else:
            token = headers

    response = api_get(base,headers=token,default_header=True)
    try:
        response = json.loads(response)
    except:
        # If the call fails, give the user a list of acceptable tags
        tags = get_tags(namespace=namespace,
                        repo_name=repo_name,
                        registry=registry,
                        auth=auth)
        print("\n".join(tags))
        logger.error("Error getting manifest for %s/%s:%s, exiting.", namespace,
                                                                       repo_name,
                                                                       repo_tag)
        print("Error getting manifest for %s/%s:%s. Acceptable tags are listed above." %(namespace,repo_name,repo_tag))
        sys.exit(1)

    return response


def get_config(manifest,spec="Entrypoint"):
    '''get_config returns a particular spec (default is Entrypoint) from a manifest obtained with get_manifest.
    :param manifest: the manifest obtained from get_manifest
    :param spec: the key of the spec to return, default is "Entrypoint"
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
        cmd = "\n".join(cmd)
    logger.info("Found Docker command (%s) %s",spec,cmd)
    return cmd


def get_layer(image_id,namespace,repo_name,download_folder=None,registry=None,auth=None):
    '''get_layer will download an image layer (.tar.gz) to a specified download folder.
    :param image_id: the (full) image id to get the manifest for, required
    :param namespace: the namespace (eg, "library")
    :param repo_name: the repo name, (eg, "ubuntu")
    :param download_folder: if specified, download to folder. Otherwise return response with raw data (not recommended)
    :param registry: the docker registry to use (default will use index.docker.io)
    :param auth: authorization header (default None)
    '''
    if registry == None:
        registry = api_base
    registry = add_http(registry) # make sure we have a complete url

    # The <name> variable is the namespace/repo_name
    base = "%s/%s/%s/%s/blobs/%s" %(registry,api_version,namespace,repo_name,image_id)
    logger.info("Downloading layers from %s", base)
    
    # To get the image layers, we need a valid token to read the repo
    token = get_token(registry=registry,
                      repo_name=repo_name,
                      namespace=namespace,
                      auth=auth)

    if download_folder != None:
        download_folder = "%s/%s.tar.gz" %(download_folder,image_id)

        # Update user what we are doing
        print("Downloading layer %s" %image_id)

    try:
        # Create temporary file with format .tar.gz.tmp.XXXXX
        fd, tmp_file = tempfile.mkstemp(prefix=("%s.tmp." % download_folder))
        os.close(fd)
        response = api_get(base,headers=token,stream=tmp_file)
        if isinstance(response, HTTPError):
            logger.error("Error downloading layer %s, exiting.", base)
            sys.exit(1)
        os.rename(tmp_file, download_folder)
    except:
        logger.error("Removing temporary download file %s", tmp_file)
        try:
            os.remove(tmp_file)
        except:
            pass
        sys.exit(1)

    return download_folder


# Under Development! ---------------------------------------------------------------------------------
# Docker Registry Version 2.0 functions

# TODO: this will let us get all Docker repos to generate images automatically
def get_repositories():
    base = "https://index.docker.io/v2/_catalog"
