#!/usr/bin/env python

'''

docker.py: Docker helper functions for Singularity in Python

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
import json

api_base = "registry-1.docker.io"
api_version = "v2"

# Authentication not required ---------------------------------------------------------------------------------

def create_runscript(cmd,base_dir):
    '''create_runscript will write a bash script with command "cmd" into the base_dir
    :param cmd: the command to write into the bash script
    :param base_dir: the base directory to write the runscript to
    '''
    runscript = "%s/singularity" %(base_dir)
    content = "#!/bin/sh\n\n%s" %(cmd)
    output_file = write_file(runscript,content)
    return output_file


def get_token(repo_name,namespace="library",scope="repository",permission="pull"):
    '''get_token will use version 2.0 of Docker's service to return a token with given permission and scope - this
    function does work, but the token doesn't seem to work when used with other functions below for authentication
    :param repo_name: the name of the repo, eg "ubuntu"
    :param repo_tag: the name of a tag for the repo, default is "latest"
    :param scope: scope of the request, default is "repository"
    :param permission: permission for the request, default is "read"
    :: note
            # https://docs.docker.com/registry/spec/auth/token/
    '''

    base = "https://auth.docker.io/token?service=registry.docker.io&scope=%s:%s/%s:%s" %(scope,
                                                                                         namespace,
                                                                                         repo_name,
                                                                                         permission)
    response = api_get(base,default_header=False)
    try:
        token = json.loads(response)["token"]
        token = {"Authorization": "Bearer %s" %(token) }
        return token
    except:
        print("Error getting %s token for repository %s/%s, exiting." %(permission,namespace,repo_name))
        sys.exit(1)



# Authentication required ---------------------------------------------------------------------------------
# Docker Registry Version 2.0 Functions - IN USE


def get_images(repo_name=None,namespace=None,manifest=None,repo_tag="latest",registry=None,auth=True):
    '''get_images is a wrapper for get_manifest, but it additionally parses the repo_name and tag's
    images and returns the complete ids
    :param repo_name: the name of the repo, eg "ubuntu"
    :param namespace: the namespace for the image, default is "library"
    :param repo_tag: the repo tag, default is "latest"
    :param registry: the docker registry url, default will use registry-1.docker.io
    '''

    # Get full image manifest, using version 2.0 of Docker Registry API
    if manifest == None:
        if repo_name != None and namespace != None:
            manifest = get_manifest(repo_name=repo_name,
                                    namespace=namespace,
                                    repo_tag=repo_tag,
                                    registry=registry,
                                    auth=auth)
        else:
            print("You must specify a namespace and repo name OR provide a manifest.")
            sys.exit(1)

    digests = []
    if 'fsLayers' in manifest:
        for fslayer in manifest['fsLayers']:
            if 'blobSum' in fslayer:
                digests.append(fslayer['blobSum'])
    return digests
    

def get_tags(namespace,repo_name,registry=None,auth=True):
    '''get_tags will return the tags for a repo using the Docker Version 2.0 Registry API
    :param namespace: the namespace (eg, "library")
    :param repo_name: the name for the repo (eg, "ubuntu")
    :param registry: the docker registry to use (default will use registry-1.docker.io
    :param auth: does the API require obtaining an authentication token? (default True)
    '''
    if registry == None:
        registry = api_base
    registry = add_http(registry) # make sure we have a complete url

    base = "%s/%s/%s/%s/tags/list" %(registry,api_version,namespace,repo_name)

    # Does the api need an auth token?
    token = None
    if auth == True:
        token = get_token(repo_name=repo_name,
                          namespace=namespace,
                          permission="pull")
       
    response = api_get(base,headers=token)
    try:
        response = json.loads(response)
        return response['tags']
    except:
        print("Error getting tags using url %s" %(base))
        sys.exit(1)


def get_manifest(repo_name,namespace,repo_tag="latest",registry=None,auth=True):
    '''get_manifest should return an image manifest for a particular repo and tag. The token is expected to
    be from version 2.0 (function above)
    :param repo_name: the name of the repo, eg "ubuntu"
    :param namespace: the namespace for the image, default is "library"
    :param repo_tag: the repo tag, default is "latest"
    :param registry: the docker registry to use (default will use registry-1.docker.io
    :param auth: does the API require obtaining an authentication Token? (default True)
    '''
    if registry == None:
        registry = api_base
    registry = add_http(registry) # make sure we have a complete url

    base = "%s/%s/%s/%s/manifests/%s" %(registry,api_version,namespace,repo_name,repo_tag)
    
    # Format the token, and prepare a header
    token = None
    if auth == True:
        token = get_token(repo_name=repo_name,
                          namespace=namespace,
                          permission="pull")

    response = api_get(base,headers=token,default_header=True)
    try:
        response = json.loads(response)
    except:
        # If the call fails, give the user a list of acceptable tags
        tags = get_tags(namespace=namespace,
                        repo_name=repo_name,
                        registry=registry)
        print("\n".join(tags))
        print("Error getting manifest for %s/%s:%s. Acceptable tags are listed above." %(namespace,repo_name,repo_tag))
        sys.exit(1)

    return response


def get_config(manifest,spec="Cmd"):
    '''get_config returns a particular spec (default is Cmd) from a manifest obtained with get_manifest.
    :param manifest: the manifest obtained from get_manifest
    :param spec: the key of the spec to return, default is "Cmd"
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
    return cmd


def get_layer(image_id,namespace,repo_name,download_folder=None,registry=None,auth=True):
    '''get_layer will download an image layer (.tar.gz) to a specified download folder.
    :param image_id: the (full) image id to get the manifest for, required
    :param namespace: the namespace (eg, "library")
    :param repo_name: the repo name, (eg, "ubuntu")
    :param download_folder: if specified, download to folder. Otherwise return response with raw data (not recommended)
    :param registry: the docker registry to use (default will use registry-1.docker.io
    :param auth: does the API require obtaining an authentication Token? (default True)
    '''
    if registry == None:
        registry = api_base
    registry = add_http(registry) # make sure we have a complete url

    # The <name> variable is the namespace/repo_name
    base = "%s/%s/%s/%s/blobs/%s" %(registry,api_version,namespace,repo_name,image_id)
    
    # To get the image layers, we need a valid token to read the repo
    token = None
    if auth == True:
        token = get_token(repo_name=repo_name,
                          namespace=namespace,
                          permission="pull")

    if download_folder != None:
        download_folder = "%s/%s.tar.gz" %(download_folder,image_id)
  
        # Update user what we are doing
        print("Downloading layer: %s" %(image_id))

    return api_get(base,headers=token,stream=download_folder)
    


# Under Development! ---------------------------------------------------------------------------------
# Docker Registry Version 2.0 functions

# TODO: this will let us get all Docker repos to generate images automatically
def get_repositories():
    base = "https://registry-1.docker.io/v2/_catalog"
