#!/usr/bin/env python
#
# Copyright (c) 2017, SingularityWare, LLC. All rights reserved.
# Copyright (c) 2015-2017, Gregory M. Kurtzer. All rights reserved.
# Copyright (c) 2017, Vanessa Sochat. All rights reserved.
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
