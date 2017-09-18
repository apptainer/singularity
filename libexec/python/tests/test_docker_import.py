'''

test_docker_import_user.py: Docker testing functions for Singularity in Python

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
import sys
sys.path.append('..')  # noqa

from subprocess import (
    Popen,
    PIPE,
    STDOUT
)

from unittest import TestCase
import shutil
import tempfile


VERSION = sys.version_info[0]

print("*** PYTHON VERSION %s DOCKER IMPORT TESTING START ***" % VERSION)


class TestImport(TestCase):

    def setUp(self):
        self.tmpdir = tempfile.mkdtemp()
        self.contents_file = "%s/hello-kitty" % self.tmpdir
        self.here = os.path.abspath(os.path.join(os.path.dirname(__file__),
                                                 os.path.pardir))

        # Variables are obtained from environment
        os.environ["SINGULARITY_CONTAINER"] = "docker://ubuntu:latest"
        os.environ["SINGULARITY_CONTENTS"] = self.contents_file

        print("\n---START----------------------------------------")

    def tearDown(self):
        shutil.rmtree(self.tmpdir)
        print("---END------------------------------------------")

    def test_IMPORT(self):
        '''test_import will test the IMPORT USER function
        '''
        script_path = "%s/import.py" % self.here
        if VERSION == 2:
            testing_command = ["python2", script_path]
        else:
            testing_command = ["python3", script_path]

        output = Popen(testing_command, stderr=STDOUT, stdout=PIPE)
        t = output.communicate()[0], output.returncode
        result = {'message': t[0],
                  'return_code': t[1]}
        self.assertEqual(result['return_code'], 0)
        self.assertTrue(os.path.exists(self.contents_file))


if __name__ == '__main__':
    unittest.main()
