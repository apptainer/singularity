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

from message import SingularityMessage
from sutils import read_file
from unittest import TestCase
import shutil
import tempfile

VERSION = sys.version_info[0]

print("*** PYTHON VERSION %s BASE TESTING START ***" % VERSION)


class TestMessage(TestCase):

    def setUp(self):
        self.logfile = tempfile.mktemp()
        
    def tearDown(self):
        os.remove(self.logfile)

    def test_logger(self):
        print('Testing message not written to file')
        bot = SingularityMessage()
        self.assertTrue(bot.logfile==None)
        bot.debug("This is a message log, not a message dog.")
        self.assertTrue(os.path.exists(self.logfile)==False)

        print('Testing that message is written to created logfile')
        os.environ['SINGULARITY_LOGFILE'] = self.logfile
        bot = SingularityMessage()
        self.assertEqual(bot.logfile, self.logfile)
        message = 'This is a message log, not a message dog.'
        bot.debug(message)
        self.assertTrue(os.path.exists(self.logfile))
        content = read_file(bot.logfile)
        self.assertTrue("DEBUG %s\n" % message in content) 

        print('Testing that message is appended to existing logfile')
        message = 'Line number two.'
        bot.info(message)
        content = read_file(bot.logfile)   
        self.assertTrue("%s\n" % message in content)


if __name__ == '__main__':
    unittest.main()
