'''

test_base.py: Test client initialized with python modules

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

import os
import re
import sys
import tarfile
sys.path.append('..')  # noqa

from unittest import TestCase
from base import ApiConnection
import shutil
import tempfile

VERSION = sys.version_info[0]

print("*** PYTHON VERSION %s BASE TESTING START ***" % VERSION)


class TestShell(TestCase):

    def setUp(self):

        self.client = ApiConnection()
        print("\n---START----------------------------------------")

    def tearDown(self):
        print("---END------------------------------------------")

    def test_client_headers(self):
        '''test_load_client will load an empty client
        '''
        print("Testing client default headers")
        required = ['Accept', 'Content-Type']
        for required_header in required:
            self.assertTrue(required_header in self.client.headers)
        self.assertEqual(self.client.update_token(), None)


if __name__ == '__main__':
    unittest.main()
