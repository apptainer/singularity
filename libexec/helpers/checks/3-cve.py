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


import os
import platform
import sys
import re
try:
    from urllib.request import urlopen, Request, unquote
except ImportError:
    from urllib2 import urlopen, Request, HTTPError


os_base, os_name, os_version = platform.linux_distribution()
os_base = os_base.lower()
os_names = "|".join([x.lower() for x in os_name.split('/')])
base = os.environ["SINGULARITY_ROOTFS"]
os.chdir(base)

##################################################################
# Common Vulnerabilities Database, High Risk
##################################################################

base = "https://security-tracker.debian.org/tracker/status/release"
filters = "?filter=1&filter=high_urgency"
release = "stable"

url = Request('%s/%s/%s' % (base, release, filters))
response = urlopen(url).read().decode('utf-8')
cve_codes = re.findall(">CVE-(.*?)<", response)

returncode = 0

# We are only testing debian
if os_base not in ['debian', 'ubuntu']:
    print("OS not in debian/ubuntu family, skipping test.")
    sys.exit(returncode)

# Iterate through the CVE codes, and assess if the distribution matches
print("Checking %s system for %s CVE vulnerabilities..." % (os_base,
                                                            len(cve_codes)))
for cve_code in cve_codes:

    url = "https://security-tracker.debian.org/tracker/CVE-%s" % cve_code
    request = Request(url)
    try:
        response = urlopen(request)
    except HTTPError:
        pass

    html = response.read().decode('utf-8')
    table = html.replace('PTS', '').split('<table>')[2]
    title = table.split('<tr>')[2]
    title = re.findall('">(.*?)</a>', title)[0]

    print("CVE-%s: %s" % (cve_code, title))

    rows = table.replace('</td>', '').split('<tr>')
    for row in rows:
        if row:
            if re.search(os_names, row):
                print("PROBLEM:  Vulnerability CVE-%s" % cve_code)
                print("RESOLVE:  %s" % url)
                returncode = 1

sys.exit(returncode)
