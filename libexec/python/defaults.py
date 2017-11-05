'''

defaults.py: this script acts as a gateway between variables defined at
runtime, and defaults. Any variable that has an unchanging default value
can be found here. The order of operations works as follows:

    1. First preference goes to environment variable set at runtime
    2. Second preference goes to default defined in this file
    3. Then, if neither is found, null is returned except in the
       case that required = True. A required = True variable not found
       will system exit with an error.

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

from message import bot
import tempfile
import os
import pwd
import sys


def getenv(variable_key, default=None, required=False, silent=False):
    '''getenv will attempt to get an environment variable. If the
    variable is not found, None is returned.
    :param variable_key: the variable name
    :param required: exit with error if not found
    :param silent: Do not print debugging information for variable
    '''
    variable = os.environ.get(variable_key, default)
    if variable is None and required:
        bot.error("Cannot find environment variable %s, exiting."
                  % (variable_key))
        sys.exit(1)

    if silent:
        if variable is not None:
            bot.verbose2("%s found" % (variable_key))
    else:
        if variable is not None:
            bot.verbose2("%s found as %s" % (variable_key, variable))
        else:
            bot.verbose2("%s not defined (None)" % (variable_key))

    return variable


def convert2boolean(arg):
    '''convert2boolean is used for environmental variables
    that must be returned as boolean'''
    if not isinstance(arg, bool):
        return arg.lower() in ("yes", "true", "t", "1", "y")
    return arg


#######################################################################
# Singularity
#######################################################################

# Filled in to exec %s "$@"
RUNSCRIPT_COMMAND_ASIS = convert2boolean(getenv("SINGULARITY_COMMAND_ASIS",
                                         default=False))

SINGULARITY_ROOTFS = getenv("SINGULARITY_ROOTFS")
METADATA_FOLDER_NAME = ".singularity.d"
_metadata_base = "%s/%s" % (SINGULARITY_ROOTFS, METADATA_FOLDER_NAME)
METADATA_BASE = getenv("SINGULARITY_METADATA_FOLDER", _metadata_base,
                       required=True)


#######################################################################
# Plugins and Formatting
#######################################################################

PLUGIN_FIXPERMS = convert2boolean(getenv("SINGULARITY_FIX_PERMS", False))

COLORIZE = getenv("SINGULARITY_COLORIZE", None)
if COLORIZE is not None:
    COLORIZE = convert2boolean(COLORIZE)

#######################################################################
# Cache
#######################################################################

DISABLE_CACHE = convert2boolean(getenv("SINGULARITY_DISABLE_CACHE",
                                default=False))

if DISABLE_CACHE is True:
    SINGULARITY_CACHE = tempfile.mkdtemp()
else:
    userhome = pwd.getpwuid(os.getuid())[5]
    _cache = os.path.join(userhome, ".singularity")
    SINGULARITY_CACHE = getenv("SINGULARITY_CACHEDIR", default=_cache)


#######################################################################
# Docker
#######################################################################

# API
DOCKER_API_BASE = "index.docker.io"  # registry
CUSTOM_REGISTRY = getenv("REGISTRY")
NAMESPACE = "library"
CUSTOM_NAMESPACE = getenv('NAMESPACE')
DOCKER_API_VERSION = "v2"
DOCKER_ARCHITECTURE = getenv("SINGULARITY_DOCKER_ARCHITECTURE", "amd64")
DOCKER_OS = getenv("SINGULARITY_DOCKER_OS", "linux")
TAG = "latest"

# Container Metadata
DOCKER_NUMBER = 10  # number to start docker files at in ENV_DIR
DOCKER_PREFIX = "docker"
SHUB_PREFIX = "shub"

# Defaults for environment, runscript, labels
_envbase = "%s/env" % (METADATA_BASE)
_runscript = "%s/singularity" % (SINGULARITY_ROOTFS)
_environment = "%s/90-environment.sh" % (_envbase)
_labelfile = "%s/labels.json" % (METADATA_BASE)
_helpfile = "%s/runscript.help" % (METADATA_BASE)
_deffile = "%s/Singularity" % (METADATA_BASE)
_testfile = "%s/test" % (METADATA_BASE)


ENVIRONMENT = getenv("SINGULARITY_ENVIRONMENT", _environment)
RUNSCRIPT = getenv("SINGULARITY_RUNSCRIPT", _runscript)
TESTFILE = getenv("SINGULARITY_TESTFILE", _testfile)
DEFFILE = getenv("SINGULARITY_DEFFILE", _deffile)
HELPFILE = getenv("SINGULARITY_HELPFILE", _helpfile)
ENV_BASE = getenv("SINGULARITY_ENVBASE", _envbase)
LABELFILE = getenv("SINGULARITY_LABELFILE", _labelfile)
INCLUDE_CMD = convert2boolean(getenv("SINGULARITY_INCLUDECMD", False))
DISABLE_HTTPS = convert2boolean(getenv("SINGULARITY_NOHTTPS", False))

#######################################################################
# Singularity Hub
#######################################################################

SINGULARITY_PULLFOLDER = getenv("SINGULARITY_PULLFOLDER", os.getcwd())
SHUB_API_BASE = "www.singularity-hub.org"
SHUB_NAMEBYHASH = getenv("SHUB_NAMEBYHASH")
SHUB_NAMEBYCOMMIT = getenv("SHUB_NAMEBYCOMMIT")
SHUB_CONTAINERNAME = getenv("SHUB_CONTAINERNAME")

#######################################################################
# Python Internal API URI Handling
#######################################################################

_layerfile = "%s/.layers" % (METADATA_BASE)
LAYERFILE = getenv("SINGULARITY_CONTENTS", _layerfile)

SINGULARITY_WORKERS = int(getenv("SINGULARITY_PYTHREADS", 9))
