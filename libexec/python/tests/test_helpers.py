'''

test_helpers.py: Helpers testing functions for Singularity in Python

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
import sys
sys.path.append('..') # directory with helpers

from unittest import TestCase
import shutil
import tempfile

from subprocess import (
    Popen,
    PIPE,
    STDOUT
)

VERSION = sys.version_info[0]

print("*** PYTHON VERSION %s UTIL HELPERS TESTING START ***" %(VERSION))

class TestJson(TestCase):


    def setUp(self):
        self.tmpdir = tempfile.mkdtemp()
        self.here = os.path.abspath(os.path.join(os.path.dirname(__file__), os.path.pardir))        
        self.file = "%s/meatballs.json" %(self.tmpdir)
        
        print("\n---START----------------------------------------")

    def tearDown(self):
        shutil.rmtree(self.tmpdir)
        print("---END------------------------------------------")
         

    def test_docker_size(self):
        '''test the function to return Docker size
        '''
        print('Testing Docker Size')
        from sutils import read_file

        os.environ['SINGULARITY_CONTAINER'] = "docker://ubuntu:latest"
        os.environ['SINGULARITY_CONTENTS'] =  self.file
        
        script_path = "%s/size.py" %(self.here)
        if VERSION == 2:
            testing_command = ["python2",script_path]
        else:
            testing_command = ["python3",script_path]

        output = Popen(testing_command,stderr=STDOUT,stdout=PIPE)
        t = output.communicate()[0],output.returncode
        result = {'message':t[0],
                  'return_code':t[1]}
        self.assertEqual(result['return_code'],0)
        result = read_file(self.file)[0]
        self.assertEqual('3621',result)


    def test_shub_size(self):
        '''test the function to return Singularity Hub Image Size
        '''
        print('Testing Singularity Hub Size')
        from sutils import read_file

        os.environ['SINGULARITY_CONTAINER'] = "shub://vsoch/singularity-hello-world"
        os.environ['SINGULARITY_CONTENTS'] =  self.file
        
        script_path = "%s/size.py" %(self.here)
        if VERSION == 2:
            testing_command = ["python2",script_path]
        else:
            testing_command = ["python3",script_path]

        output = Popen(testing_command,stderr=STDOUT,stdout=PIPE)
        t = output.communicate()[0],output.returncode
        result = {'message':t[0],
                  'return_code':t[1]}
        self.assertEqual(result['return_code'],0)
        result = read_file(self.file)[0]
        self.assertEqual('331',result)


if __name__ == '__main__':
    unittest.main()
