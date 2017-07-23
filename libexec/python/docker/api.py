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
import math
import os
sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__),
                                os.path.pardir)))  # noqa
sys.path.append('..')  # noqa

from base import ApiConnection

from sutils import (
    add_http,
    get_cache,
    change_tar_permissions,
    create_tar,
    write_singularity_infos
)

from defaults import (
    API_BASE,
    API_VERSION,
    DOCKER_NUMBER,
    DOCKER_PREFIX,
    ENV_BASE,
    LABELFILE,
    METADATA_FOLDER_NAME,
    RUNSCRIPT_COMMAND_ASIS
)

from helpers.json.main import ADD
from message import bot
from shell import parse_image_uri
from templates import get_template

import json
import re
import tempfile
try:
    from urllib.error import HTTPError
except ImportError:
    from urllib2 import HTTPError


# Docker API Class  ---------------------------------------

class DockerApiConnection(ApiConnection):

    def __init__(self, **kwargs):
        self.auth = None
        self.token = None
        self.token_url = None
        self.api_base = API_BASE
        self.api_version = API_VERSION
        self.manifest = None

        if 'auth' in kwargs:
            self.auth = kwargs['auth']
        if 'token' in kwargs:
            self.token = kwargs['token']
        super(DockerApiConnection, self).__init__(**kwargs)
        if 'image' in kwargs:
            self.load_image(kwargs['image'])

    def assemble_uri(self, sep=None):
        '''re-assemble the image uri, for components defined.
        '''
        if sep is None:
            sep = "-"
        image_uri = "%s%s%s%s%s" % (self.registry, sep,
                                    self.namespace, sep,
                                    self.repo_name)

        if self.version is not None:
            image_uri = "%s@%s" % (image_uri, self.version)
        else:
            image_uri = "%s:%s" % (image_uri, self.repo_tag)

        return image_uri

    def _init_headers(self):
        # specify wanting version 2 schema
        # meaning the correct order of digests
        # returned (base to child)

        return {"Accept": 'application/vnd.docker.distribution.manifest.v2+json,application/vnd.docker.distribution.manifest.list.v2+json',  # noqa
               'Content-Type': 'application/json; charset=utf-8'}  # noqa

    def check_errors(self, response, exit=True):
        '''take a response with errors key,
        iterate through errors in expected format and exit upon completion
        '''
        if "errors" in response:
            for error in response['errors']:
                bot.error("%s: %s" % (error['code'],
                                      error['message']))
                if error['code'] == "UNAUTHORIZED":
                    msg = "Check existence, naming, and permissions"
                    bot.error(msg)
            if exit:
                sys.exit(1)
        return response

    def load_image(self, image):
        '''load_image parses the image uri, and loads the
        different image parameters into the client.
        The image should be a docker uri (eg docker://)
        or name of docker image'''

        image = parse_image_uri(image=image, uri="docker://")
        self.repo_name = image['repo_name']
        self.repo_tag = image['repo_tag']
        self.namespace = image['namespace']
        self.version = image['version']
        self.registry = image['registry']
        self.update_token()

    def update_token(self, response=None, auth=None):
        '''update_token uses HTTP basic authentication to get a token for
        Docker registry API V2 operations. We get here if a 401 is
        returned for a request.
        https://docs.docker.com/registry/spec/auth/token/
        '''
        if self.token_url is None:

            if response is None:
                response = self.get_tags(return_response=True)

            if not isinstance(response, HTTPError):
                bot.verbose3('Response on obtaining token is None.')
                return None

            not_asking_auth = "Www-Authenticate" not in response.headers
            if response.code != 401 or not_asking_auth:
                bot.error("Authentication error, exiting.")
                sys.exit(1)

            challenge = response.headers["Www-Authenticate"]
            regexp = '^Bearer\s+realm="(.+)",service="(.+)",scope="(.+)",?'
            match = re.match(regexp, challenge)

            if not match:
                bot.error("Unrecognized authentication challenge, exiting.")
                sys.exit(1)

            realm = match.group(1)
            service = match.group(2)
            scope = match.group(3).split(',')[0]

            self.token_url = ("%s?service=%s&expires_in=9000&scope=%s"
                              % (realm, service, scope))

        headers = dict()

        # First priority comes to auth supplied directly to function
        if auth is not None:
            headers.update(auth)

        # Second priority is default if supplied at init
        elif self.auth is not None:
            headers.update(self.auth)

        response = self.get(self.token_url,
                            default_headers=False,
                            headers=headers)

        try:
            token = json.loads(response)["token"]
            token = {"Authorization": "Bearer %s" % token}
            self.token = token
            self.update_headers(token)

        except Exception:
            msg = "Error getting token for repository "
            msg += "%s/%s, exiting." % (self.namespace,
                                        self.repo_name)
            bot.error(msg)
            sys.exit(1)

    def get_images(self):
        '''get_images is a wrapper for get_manifest, but it
        additionally parses the repo_name and tag's images
        and returns the complete ids
        :param repo_name: the name of the repo, eg "ubuntu"
        :param namespace: the namespace for the image
                          default is "library"
        :param repo_tag: the repo tag default "latest"
        :param registry: the docker registry url
                         default will use index.docker.io
        '''

        # Get full image manifest, using version 2.0 of Docker Registry API
        if self.manifest is None:
            if self.repo_name is not None and self.namespace is not None:
                self.manifest = self.get_manifest()

            else:
                bot.error("No namespace or sufficient metadata to get one.")
                sys.exit(1)

        digests = read_digests(self.manifest)
        return digests

    def get_tags(self, return_response=False):
        '''get_tags will return the tags for a repo using the
        Docker Version 2.0 Registry API
        '''
        registry = self.registry
        if registry is None:
            registry = self.api_base

        registry = add_http(registry)  # make sure we have a complete url

        base = "%s/%s/%s/%s/tags/list" % (registry,
                                          self.api_version,
                                          self.namespace,
                                          self.repo_name)

        bot.verbose("Obtaining tags: %s" % base)

        # We use get_tags for a testing endpoint in update_token
        response = self.get(base,
                            return_response=return_response)

        if return_response:
            return response

        try:
            response = json.loads(response)
            return response['tags']
        except Exception:
            bot.error("Error obtaining tags: %s" % base)
            sys.exit(1)

    def get_manifest(self, old_version=False):
        '''get_manifest should return an image manifest
        for a particular repo and tag.  The image details
        are extracted when the client is generated.
        :param old_version: return version 1
                            (for cmd/entrypoint), default False
        '''
        registry = self.registry
        if registry is None:
            registry = self.api_base

        # make sure we have a complete url
        registry = add_http(registry)

        base = "%s/%s/%s/%s/manifests" % (registry,
                                          self.api_version,
                                          self.namespace,
                                          self.repo_name)
        if self.version is not None:
            base = "%s/%s" % (base, self.version)
        else:
            base = "%s/%s" % (base, self.repo_tag)
        bot.verbose("Obtaining manifest: %s" % base)

        headers = self.headers
        if old_version is True:
            headers['Accept'] = 'application/json'

        response = self.get(base, headers=self.headers)

        try:
            response = json.loads(response)

        except Exception:

            # If the call fails, give the user a list of acceptable tags
            tags = self.get_tags()
            print("\n".join(tags))
            repo_uri = "%s/%s:%s" % (self.namespace,
                                     self.repo_name,
                                     self.repo_tag)

            bot.error("Error getting manifest for %s, exiting." % repo_uri)
            sys.exit(1)

        # If we have errors, don't continue
        return self.check_errors(response)

    def get_layer(self,
                  image_id,
                  download_folder=None,
                  change_perms=False,
                  return_tmp=False):

        '''get_layer will download an image layer (.tar.gz)
        to a specified download folder.
        :param download_folder: if specified, download to folder.
                                Otherwise return response with raw data
        :param change_perms: change permissions additionally
                             (default False to support multiprocessing)
        :param return_tmp: If true, return the temporary file name (and
                             don't rename to the file's final name). Default
                             is False, should be True for multiprocessing
                             that requires extra permission changes
        '''
        registry = self.registry
        if registry is None:
            registry = self.api_base

        # make sure we have a complete url
        registry = add_http(registry)

        # The <name> variable is the namespace/repo_name
        base = "%s/%s/%s/%s/blobs/%s" % (registry,
                                         self.api_version,
                                         self.namespace,
                                         self.repo_name,
                                         image_id)
        bot.verbose("Downloading layers from %s" % base)

        if download_folder is None:
            download_folder = tempfile.mkdtemp()

        download_folder = "%s/%s.tar.gz" % (download_folder, image_id)

        # Update user what we are doing
        bot.debug("Downloading layer %s" % image_id)

        # Step 1: Download the layer atomically
        file_name = "%s.%s" % (download_folder,
                               next(tempfile._get_candidate_names()))
        tar_download = self.download_atomically(url=base,
                                                file_name=file_name)
        bot.debug('Download of raw file (pre permissions fix) is %s'
                  % tar_download)

        # Step 2: Fix Permissions?
        if change_perms:
            tar_download = change_tar_permissions(tar_download)

        if return_tmp is True:
            return tar_download

        try:
            os.rename(tar_download, download_folder)
        except Exception:
            msg = "Cannot untar layer %s," % tar_download
            msg += " was there a problem with download?"
            bot.error(msg)
            sys.exit(1)
        return download_folder

    def get_size(self, add_padding=True, round_up=True, return_mb=True):
        '''get_size will return the image size (must use v.2.0 manifest)
        :add_padding: if true, return reported size * 5
        :round_up: if true, round up to nearest integer
        :return_mb: if true, defaults bytes are converted to MB
        '''
        manifest = self.get_manifest()
        size = None
        if "layers" in manifest:
            size = 0
            for layer in manifest["layers"]:
                if "size" in layer:
                    size += layer['size']

            if add_padding is True:
                size = size * 5

            if return_mb is True:
                size = size / (1024 * 1024)  # 1MB = 1024*1024 bytes

            if round_up is True:
                size = math.ceil(size)
            size = int(size)

        return size

    def get_config(self, spec="Entrypoint", delim=None, old_version=False):
        '''get_config returns a particular spec (default is Entrypoint)
        from a VERSION 1 manifest obtained with get_manifest.
        :param manifest: the manifest obtained from get_manifest
        :param spec: the key of the spec to return, default is "Entrypoint"
        :param delim: Given a list, the delim to use to join the entries.
                      Default is newline
        '''
        manifest = self.get_manifest(old_version=old_version)

        cmd = None

        # Version 1 of the manifest has more detailed metadata
        if old_version:
            if "history" in manifest:
                for entry in manifest['history']:
                    if 'v1Compatibility' in entry:
                        entry = json.loads(entry['v1Compatibility'])
                        if "config" in entry:
                            if spec in entry["config"]:
                                cmd = entry["config"][spec]

            # Standard is to include commands like ['/bin/sh']
            if isinstance(cmd, list):
                if delim is None:
                    delim = "\n"
                cmd = delim.join(cmd)
            bot.verbose("Found Docker config (%s) %s" % (spec, cmd))

        else:
            if "config" in manifest:
                if spec in manifest['config']:
                    cmd = manifest['config'][spec]
        return cmd


# API Helper functions

def read_digests(manifest):
    '''read_layers will return a list of layers from a manifest.
    The function is intended to work with both version
    1 and 2 of the schema
    :param manifest: the manifest to read_layers from
    '''

    digests = []

    # https://github.com/docker/distribution/blob/master/docs/spec/manifest-v2-2.md#image-manifest  # noqa
    if 'layers' in manifest:
        layer_key = 'layers'
        digest_key = 'digest'
        bot.debug('Image manifest version 2.2 found.')

    # https://github.com/docker/distribution/blob/master/docs/spec/manifest-v2-1.md#example-manifest  # noqa
    elif 'fsLayers' in manifest:
        layer_key = 'fsLayers'
        digest_key = 'blobSum'
        bot.debug('Image manifest version 2.1 found.')

    else:
        msg = "Improperly formed manifest, "
        msg += "layers or fsLayers must be present"
        bot.error(msg)
        sys.exit(1)

    for layer in manifest[layer_key]:
        if digest_key in layer:
            if layer[digest_key] not in digests:
                bot.debug("Adding digest %s" % layer[digest_key])
                digests.append(layer[digest_key])
    return digests
