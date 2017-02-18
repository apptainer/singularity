'''

defaults.py: this script acts as a gateway between variables defined at
runtime, and defaults. Any variable that has an unchanging default value 
can be found here, for example, the folder at the root of the container for
metadata (METADATA_BASE) is consistent. A variable like SINGULARITY_ROOTFS,
on the other hand, is not, and is not in this file. The order of operations
works as follows:
  
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
import os

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
        logger.debug("%s found as %s",variable_key,variable)

    return variable 


#######################################################################
# Singularity
#######################################################################

_cache = os.path.join(os.environ.get("HOME"),".singularity") 
SINGULARITY_CACHE = getenv("SINGULARITY_CACHEDIR", default=_cache)

METADATA_BASE = getenv("SINGULARITY_METADATA_FOLDER", 
                       default="/.singularity-info",
                       required=True)

DISABLE_CACHE = getenv("SINGULARITY_DISABLE_CACHE",
                       default=False)


#######################################################################
# Docker
#######################################################################

# API
API_VERSION = "v2"
NAMESPACE = "library"
REGISTRY = "index.docker.io" # registry
TAG = "latest"

# Container Metadata
DOCKER_NUMBER = 10 # number to start docker files at in ENV_DIR
DOCKER_PREFIX = "docker"
ENV_BASE = ".env"
LAYERFILE = ".layers"
LABELS_BASE = ".labels"


