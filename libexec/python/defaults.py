'''

defaults.py: python helper for singularity command line tool. This script stores
default variables for core python modules for singularity, akin to __init__.py.

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

import os

#######################################################################
# Singularity
#######################################################################

SINGULARITY_CACHE = os.path.join(os.environ.get("HOME"),".singularity")
METADATA_DIR = "/.singularity-info"


#######################################################################
# Docker
#######################################################################

REGISTRY = "index.docker.io" # registry
NAMESPACE = "library"
TAG = "latest"
API_VERSION = "v2"
LAYERFILE = ".layers"
ENV_DIR = ".env"
LABEL_DIR = ".label"
