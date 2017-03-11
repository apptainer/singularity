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

from logman import logger
import tempfile
import os
import pwd
import sys

def getenv(variable_key,required=False,default=None,silent=False):
    '''getenv will attempt to get an environment variable. If the variable
    is not found, None is returned.
    :param variable_key: the variable name
    :param required: exit with error if not found
    :param silent: Do not print debugging information for variable
    '''
    variable = os.environ.get(variable_key, default)
    if variable == None and required:
        logger.error("Cannot find environment variable %s, exiting.",variable_key)
        sys.exit(1)

    if silent:
        logger.debug("%s found",variable_key)
    else:
        if variable != None:
            logger.debug("%s found as %s",variable_key,variable)
        else:
            logger.debug("%s not defined (None)",variable_key)

    return variable 

def convert2boolean(arg):
  '''convert2boolean is used for environmental variables that must be
  returned as boolean'''
  if not isinstance(arg,bool):
      return arg.lower() in ("yes", "true", "t", "1","y")
  return arg


#######################################################################
# Singularity
#######################################################################

# Filled in to exec %s "$@"
RUNSCRIPT_COMMAND_ASIS = convert2boolean(getenv("SINGULARITY_COMMAND_ASIS",
                                         default=False))

SINGULARITY_ROOTFS = getenv("SINGULARITY_ROOTFS")
_metadata_base = "%s/.singularity" %(SINGULARITY_ROOTFS)
METADATA_BASE = getenv("SINGULARITY_METADATA_FOLDER", 
                                   default=_metadata_base,
                                   required=True)

#######################################################################
# Cache
#######################################################################

DISABLE_CACHE = convert2boolean(getenv("SINGULARITY_DISABLE_CACHE",
                                default=False))

if DISABLE_CACHE == True:
    SINGULARITY_CACHE = tempfile.mkdtemp()
else:
    userhome = pwd.getpwuid(os.getuid())[5]
    _cache = os.path.join(userhome,".singularity") 
    SINGULARITY_CACHE = getenv("SINGULARITY_CACHEDIR", default=_cache)

if not os.path.exists(SINGULARITY_CACHE):
    os.mkdir(SINGULARITY_CACHE)


#######################################################################
# Docker
#######################################################################

# API
API_BASE = "index.docker.io" # registry
API_VERSION = "v2"
NAMESPACE = "library"
TAG = "latest"

# Container Metadata
DOCKER_NUMBER = 10 # number to start docker files at in ENV_DIR
DOCKER_PREFIX = "docker"
SHUB_PREFIX = "shub"

_envbase = "%s/env" %(METADATA_BASE)
ENV_BASE = getenv("SINGULARITY_ENVBASE", default=_envbase)
_labelbase = "%s/labels" %(METADATA_BASE)
LABEL_BASE = getenv("SINGULARITY_LABELBASE", default=_labelbase)


#######################################################################
# Singularity Hub
#######################################################################

SINGULARITY_PULLFOLDER = getenv("SINGULARITY_PULLFOLDER", default=os.getcwd())
SHUB_API_BASE = "singularity-hub.org/api"


#######################################################################
# Python Internal API URI Handling
#######################################################################

_layerfile = "%s/.layers" %(METADATA_BASE)
LAYERFILE = getenv("SINGULARITY_CONTENTS", default=_layerfile)

#URI_IMAGE = "img://"
#URI_TAR = "tar://"
#URI_TARGC = "targz://"
