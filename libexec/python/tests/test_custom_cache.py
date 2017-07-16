'''

test_custom_cache.py: Testing custom cache function

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
import re
import sys
sys.path.append('..')  # noqa

from unittest import TestCase
import shutil
import tempfile

VERSION = sys.version_info[0]

print("*** PYTHON VERSION %s CLIENT TESTING START ***" % VERSION)


class TestCustomCache(TestCase):

    def setUp(self):
        self.tmpdir = tempfile.mkdtemp()
        self.custom_cache = '%s/cache' % self.tmpdir
        os.environ['SINGULARITY_CACHEDIR'] = self.custom_cache
        os.environ['SINGULARITY_ROOTFS'] = self.tmpdir
        print("\n---START----------------------------------------")

    def tearDown(self):
        shutil.rmtree(self.tmpdir)
        print("---END------------------------------------------")

    def test_get_cache_custom(self):
        '''test_run_command tests sending a command to commandline
        using subprocess
        '''
        print("Testing get_cache with environment set")
        from defaults import SINGULARITY_CACHE
        self.assertEqual(self.custom_cache, SINGULARITY_CACHE)


if __name__ == '__main__':
    unittest.main()
