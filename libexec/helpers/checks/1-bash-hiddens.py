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
# Alert the user about history and profiles

import platform
import sys
import os

base = os.environ["SINGULARITY_ROOTFS"]
os.chdir(base)

os_base, os_name, os_version = platform.linux_distribution()
os_base = os_base.lower()

returncode = 0

if os.geteuid() != 0:
    print("You must run this test as sudo, skipping")
    sys.exit(returncode)


def find_history(returncode):
    '''find_history will return 1 if any history files
    are found'''

    if os.path.exists('root'):
        history = [x for x in os.listdir('root')
                   if x.endswith('history') or 'hist' in x]

        # The apt cache should be cleaned
        if len(history) > 0:
            print("PROBLEM:  history at /root home found.")
            print("RESOLVE:  check for sensitive content.")
            print('\n'.join(history))
            returncode = 1

    return returncode


def find_profiles(returncode):
    '''find_profiles will return 1 if any profile files
    are found'''

    if os.path.exists('root'):
        profiles = [x for x in os.listdir('root')
                    if x.endswith('rc') or 'profile' in x]

        # The apt cache should be cleaned
        if len(profiles) > 0:
            print("PROBLEM:  profiles at /root home found.")
            print("RESOLVE:  check for sensitive content.")
            print('\n'.join(profiles))
            returncode = 1

    return returncode


# Debian Cache
if os_base in ["debian", "ubuntu", "centos", "redhat"]:

    if os.path.exists("root"):
        returncode = find_history(returncode)
        returncode = find_profiles(returncode)

sys.exit(returncode)
