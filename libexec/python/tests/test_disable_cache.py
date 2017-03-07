'''

test_disable_cache.py: Testing disabled cache function

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
sys.path.append('..') # base directory

from unittest import TestCase
import shutil
import tempfile

VERSION = sys.version_info[0]

print("*** PYTHON VERSION %s CLIENT TESTING START ***" %(VERSION))


class TestDisabledCache(TestCase):

    def setUp(self):
        os.environ['SINGULARITY_DISABLE_CACHE'] = "yes"
        self.tmpdir = tempfile.mkdtemp() 
        os.environ['SINGULARITY_ROOTFS'] = self.tmpdir
        print("\n---START----------------------------------------")

    def tearDown(self):
        shutil.rmtree(self.tmpdir)
        print("---END------------------------------------------")

    def test_cache_disabled(self):
        print("Case 3: Testing that cache gets disabled.")
        from defaults import DISABLE_CACHE
        self.assertEqual(DISABLE_CACHE,True)

if __name__ == '__main__':
    unittest.main()
