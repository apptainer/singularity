#!/usr/bin/env python
#
# Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
# Copyright (c) 2017, Vanessa Sochat. All rights reserved.
#
# See the COPYRIGHT.md file at the top-level directory of this
# distribution and at https://github.com/singularityware/singularity
#
# This file is part of the Singularity Linux container project.
# It is subject to the license terms in the LICENSE.md file
# found in the top-level directory of this distribution and at
# https://github.com/singularityware/singularity. No part
# of Singularity, including this file, may be copied, modified,
# propagated, or distributed except according to the terms
# contained in the LICENSE.md file.
#
# Yell at the user for bad practices from Docker containers

import sys
import os

base = os.environ["SINGULARITY_ROOTFS"]
os.chdir(base)

returncode = 0

if os.geteuid() != 0:
    print("You must run this test as sudo, skipping")
    sys.exit(returncode)

# Apt-get cache
skip = ['.profile', '.bashrc', '.tcshc',
        '.cshrc', '.bash_history',
        '.bash_profile']

root = [x for x in os.listdir('root') if x not in skip]

# The user should not put content in root!
if len(root) > 0:
    print("PROBLEM:  You should not save content in roots home.")
    print("RESOLVE:  Install to /opt or /usr/local")
    print("\n".join(root))
    returncode = 1

sys.exit(returncode)
