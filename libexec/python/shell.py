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
    DOCKER_API_BASE,
    NAMESPACE,
    CUSTOM_REGISTRY,
    CUSTOM_NAMESPACE,
    TAG
)

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
    '''remove_image_uri will return just the image name.
    ::note this will also remove all spaces from the uri.
    '''
    if image_uri is None:
        image_uri = get_image_uri(image, quiet=quiet)

    image = image.replace(' ', '')

    if image_uri is not None:
        image = image.replace(image_uri, '')
    return image

# Quick python regex syntax reference for referring back to matched groups:
# (?:stuff) matches stuff, just like (stuff) but won't appear in m.groups()
# (?P<name>stuff) matches stuff, and can be retrieved by m.group('name')

# Regular expression used to parse docker:// uris, which have slightly
# different rules than shub://, namely:
# - registry must include :port or a . in the name (e.g. docker.io)
# - namespace is completely optional
_docker_uri_re = re.compile(
    # Optionally match a registry if it contains a '.' or a ':',
    # may contain any character but '/' or '@', and terminated by '/'
    # Note the use of (?:) to make sure the group "registry" does
    # not contain the / at the end. Also, registry includes the port
    "(?:(?P<registry>[^/@]+[.:][^/@]*)/)?"
    # Optionally match a namespace, matched as any characters but
    # ':' or '/', followed by '/'. Note the match will include the final /
    "(?P<namespace>(?:[^:@/]+/)+)?"
    # Match a repo name, mandatory. Any character but ':' or '/'
    "(?P<repo>[^:@/]+)"
    # Match :tag (optional)
    "(?::(?P<tag>[^:@]+))?"
    # Match @digest (optional)
    "(?:@(?P<version>.+))?"
    # we need to match the whole string, make sure there's no leftover
    "$"
    )


# Reduced regex, matches registry:port/repo or registry.com/repo
# (registry must include : or .) with optional tag or version.
# Also matches repo only, e.g. reponame[:tag|@version].
# This is tried before _default_uri_re, so that namespace/repo takes
# precedence over registry/repo.
_reduced_uri_no_ns_re = re.compile(
    # match a registry, optional, may include a : or ., but not a @
    "(?:(?P<registry>[^/@]+[.:][^/@]*)/)?"
    # Match a repo name, mandatory. Any character but ':' or '/'
    "(?P<repo>[^:@/]+)"
    # Match :tag (optional)
    "(?::(?P<tag>[^:@]+))?"
    # Match @digest (optional)
    "(?:@(?P<version>.+))?"
    # we need to match the whole string, make sure there's no leftover
    "$"
    # dummy group that will never match, but will add a 'namespace' entry
    "(?P<namespace>.)?"
    )

# Regular expression used to parse other images and uris (e.g. shub://)
# This version expects a namespace, and won't match otherwise
# but the registry is optional. In cases like a/b/c, the first component
# is taken to be the registry in this regex.
_default_uri_re = re.compile(
    # must be either registry[:port]/namespace[/more/namespaces]/repo
    # or namespace/repo, at least one namespace is required
    # Match registry, if specified
    "(?:(?P<registry>[^/@]+)/)?"
    # Match a namespace, matched as any characters but
    # ':' or '@', ended by a '/'. Note the match will include the final /
    "(?P<namespace>(?:[^:@/]+/)+)"
    # Match a repo name, mandatory. Any character but ':' or '/'
    "(?P<repo>[^:@/]+)"
    # Match :tag (optional)
    "(?::(?P<tag>[^:@]+))?"
    # Match @digest (optional)
    "(?:@(?P<version>.+))?"
    # we need to match the whole string, make sure there's no leftover
    "$"
    )

def parse_image_uri(image,
                    uri=None,
                    quiet=False,
                    default_registry=None,
                    default_namespace=None,
                    default_tag=None):
    '''parse_image_uri will parse a docker or shub uri and return
    a json structure with a registry, repo name, tag, namespace and version.
    Tag and version are optional, namespace is optional for docker:// uris.
    URIs are of this general form:
        myuri://[registry.com:port/][namespace/nested/]repo[:tag][@version]
    (parts in [] are optional)
    Parsing rules are slightly different if the uri is a docker:// uri:
    - registry must include a :port or a . in its name, else will be parsed
      as a namespace. For non-docker uris, instead, if there are three or
      more parts separated by /, the first one is taken to be the registry
    - namespace can be empty if a registry is specified, else default
      namespace will be used (e.g. docker://registry.com/repo:tag).
      For non-docker uris, namespace cannot be empty and default will be used

    :param image: the string provided on command line for
                  the image name, eg: ubuntu:latest or
                  docker://local.registry/busybox@12345
    :param uri: the uri type (eg, docker://), default autodetects
    ::note uri is maintained as variable so we have some control over allowed
    :param quiet: If True, don't show verbose messages with the parsed values
    :default_registry: Which registry to use if image doesn't contain one.
                       if None, use defaults.REGISTRY
    :default_namespace: Which namespace to use if image doesn't contain one.
                       if None, use defaults.NAMESPACE. Also, check out the
                       note above about docker:// and empty namespaces.
    :default_tag: Which tag to use if image doesn't contain one.
                       if None, use defaults.REPO_TAG
    :returns parsed: a json structure with repo_name, repo_tag, and namespace
    '''

    # Default to most common use case, Docker
    if default_registry is None:
        default_registry = DOCKER_API_BASE

    if default_namespace is None:
        default_namespace = NAMESPACE

    if default_tag is None:
        default_tag = TAG

    # if user gave custom registry / namespace, make them the default
    if CUSTOM_NAMESPACE is not None:
        default_namespace = CUSTOM_NAMESPACE

    if CUSTOM_REGISTRY is not None:
        default_registry = CUSTOM_REGISTRY

    # candidate regex for matching, in order of preference
    uri_regexes = [ _reduced_uri_no_ns_re,
                    _default_uri_re ]

    # Be absolutely sure there are no comments
    image = image.split('#')[0]

    if not uri:
        uri = get_image_uri(image, quiet=True)

    # docker images require slightly different rules
    if uri == "docker://":
        uri_regexes = [ _docker_uri_re ]

    image = remove_image_uri(image, uri)

    for r in uri_regexes:
        match = r.match(image)
        if match:
            break

    if not match:
        bot.error('Could not parse image "%s"! Exiting.' % image)
        sys.exit(1)

    registry = match.group('registry')
    namespace = match.group('namespace')
    repo_name = match.group('repo')
    repo_tag = match.group('tag')
    version = match.group('version')

    if namespace:
        # strip trailing /
        namespace = namespace.rstrip('/')

    # repo_name is required and enforced by the re (in theory)
    # if not provided, re should not match
    assert(repo_name)

    # replace empty fields with defaults
    if not namespace:
        # for docker, if a registry was specified, don't
        # inject a namespace in between
        if registry and uri == "docker://":
            namespace = ""
        else:
            namespace = default_namespace
    if not registry:
        registry = default_registry
    if not repo_tag:
        repo_tag = default_tag
    # version is not mandatory, don't check that


    if not quiet:
        bot.verbose("Registry: %s" % registry)
        bot.verbose("Namespace: %s" % namespace)
        bot.verbose("Repo Name: %s" % repo_name)
        bot.verbose("Repo Tag: %s" % repo_tag)
        bot.verbose("Version: %s" % version)

    parsed = {'registry': registry,
              'namespace': namespace,
              'repo_name': repo_name,
              'repo_tag': repo_tag,
              'version': version }

    return parsed
