'''

test_json.py: Singularity Hub testing functions for Singularity in Python

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

from unittest import TestCase
import shutil
import tempfile

from subprocess import (
    Popen,
    PIPE,
    STDOUT
)

VERSION = sys.version_info[0]

print("*** PYTHON VERSION %s UTIL HELPERS TESTING START ***" % VERSION)


class TestJson(TestCase):

    def setUp(self):
        self.tmpdir = tempfile.mkdtemp()
        self.here = os.path.abspath(os.path.join(os.path.dirname(__file__),
                                                 os.path.pardir))
        self.file = "%s/meatballs.json" % self.tmpdir

        print("\n---START----------------------------------------")

    def tearDown(self):
        shutil.rmtree(self.tmpdir)
        print("---END------------------------------------------")

    def format_keyname(self):
        '''test the function to format the key name
        '''
        from helpers.json.main import format_keyname
        print("Testing formatting of key function.")

        print('Case 1: Testing that key returns all caps')
        key = format_keyname('dry_meatball')
        self.assertEqual(key, 'DRY_MEATBALL')

        print('Case 2: Testing that key replaced invalid characters with _')
        key = format_keyname('big!!meatball)#$FTW')
        self.assertEqual(key, 'DRY_MEATBALL_FTW')
        key = format_keyname('ugly-meatball')
        self.assertEqual(key, 'UGLY_MEATBALL')

    def test_get(self):
        '''test_get will test the get function
        '''
        print('Testing json GET')

        print('Case 1: Get exiting key')
        from sutils import write_json
        write_json({"PASTA": "rigatoni!"}, self.file)
        self.assertTrue(os.path.exists(self.file))

        script_path = "%s/helpers/json/get.py" % self.here
        if VERSION == 2:
            testing_command = ["python2",
                               script_path,
                               '--key',
                               'PASTA',
                               '--file',
                               self.file]
        else:
            testing_command = ["python3",
                               script_path,
                               '--key',
                               'PASTA',
                               '--file',
                               self.file]

        output = Popen(testing_command,
                       stderr=PIPE,
                       stdout=PIPE)
        t = output.communicate()[0], output.returncode
        result = {'message': t[0],
                  'return_code': t[1]}
        self.assertEqual(result['return_code'], 0)

        output = result['message']
        if isinstance(output, bytes):
            output = output.decode(encoding='UTF-8')
        self.assertEqual('rigatoni!',
                         output.strip('\n').split('\n')[-1])

        print('Case 2: Get non-existing key exits')
        if VERSION == 2:
            testing_command = ["python2", script_path, '--key',
                               'LASAGNA', '--file', self.file]
        else:
            testing_command = ["python3", script_path, '--key',
                               'LASAGNA', '--file', self.file]

        output = Popen(testing_command,
                       stderr=PIPE,
                       stdout=PIPE)
        t = output.communicate()[0], output.returncode
        result = {'message': t[0],
                  'return_code': t[1]}
        self.assertEqual(result['return_code'], 1)

    def test_add_delete(self):
        '''test_add_delete will test the add and delete functions
        '''
        print('Testing json ADD')
        from sutils import write_json, read_json

        print('Case 1: Adding to new file, force not needed')
        self.assertTrue(os.path.exists(self.file) is False)

        script_path = "%s/helpers/json/add.py" % self.here
        if VERSION == 2:
            testing_command = ["python2", script_path, '--key', 'LEGO',
                               '--value', 'RED', '--file', self.file]
        else:
            testing_command = ["python3", script_path, '--key', 'LEGO',
                               '--value', 'RED', '--file', self.file]

        output = Popen(testing_command,
                       stderr=PIPE,
                       stdout=PIPE)

        t = output.communicate()[0], output.returncode
        result = {'message': t[0],
                  'return_code': t[1]}
        self.assertEqual(result['return_code'], 0)
        self.assertTrue(os.path.exists(self.file))

        # Check the contents
        contents = read_json(self.file)
        self.assertTrue('LEGO' in contents)
        self.assertTrue(contents['LEGO'] == 'RED')

        print('Case 2: Adding to existing key without force should error.')
        output = Popen(testing_command,
                       stderr=PIPE,
                       stdout=PIPE)
        t = output.communicate()[0], output.returncode
        result = {'message': t[0],
                  'return_code': t[1]}
        self.assertEqual(result['return_code'], 1)

        print('Case 3: Adding to existing key with force should work.')
        if VERSION == 2:
            testing_command = ["python2", script_path, '--key', 'LEGO',
                               '--value', 'BLUE', '--file', self.file, '-f']
        else:
            testing_command = ["python3", script_path, '--key', 'LEGO',
                               '--value', 'BLUE', '--file', self.file, '-f']

        output = Popen(testing_command,
                       stderr=PIPE,
                       stdout=PIPE)
        t = output.communicate()[0], output.returncode
        result = {'message': t[0],
                  'return_code': t[1]}
        self.assertEqual(result['return_code'], 0)

        # Check the updated contents
        contents = read_json(self.file)
        self.assertTrue('LEGO' in contents)
        self.assertTrue(contents['LEGO'] == 'BLUE')

        if VERSION == 2:
            testing_command = ["python2", script_path, '--key', 'PASTA',
                               '--value', 'rigatoni!', '--file', self.file]
        else:
            testing_command = ["python3", script_path, '--key', 'PASTA',
                               '--value', 'rigatoni!', '--file', self.file]

        output = Popen(testing_command,
                       stderr=PIPE,
                       stdout=PIPE)
        t = output.communicate()[0], output.returncode
        result = {'message': t[0],
                  'return_code': t[1]}

        print('Case 4: Deleting key from file')
        script_path = "%s/helpers/json/delete.py" % self.here
        if VERSION == 2:
            testing_command = ["python2", script_path, '--key',
                               'LEGO', '--file', self.file]
        else:
            testing_command = ["python3", script_path,
                               '--key', 'LEGO', '--file', self.file]

        output = Popen(testing_command,
                       stderr=PIPE,
                       stdout=PIPE)
        t = output.communicate()[0], output.returncode
        result = {'message': t[0],
                  'return_code': t[1]}
        self.assertEqual(result['return_code'], 0)

        # Check the key was deleted contents
        contents = read_json(self.file)
        self.assertTrue('LEGO' not in contents)

        print('Case 5: Checking that empty file is removed.')
        if VERSION == 2:
            testing_command = ["python2", script_path, '--key',
                               'PASTA', '--file', self.file]
        else:
            testing_command = ["python3", script_path, '--key',
                               'PASTA', '--file', self.file]

        output = Popen(testing_command,
                       stderr=PIPE,
                       stdout=PIPE)
        t = output.communicate()[0], output.returncode
        result = {'message': t[0],
                  'return_code': t[1]}
        self.assertTrue(os.path.exists(self.file) is False)

    def test_dump(self):
        '''test_add_delete will test the add and delete functions
        '''
        print('Testing json DUMP')
        from sutils import write_json, read_json

        print('Case 1: Dumping file.')

        jsondump = {'HELLO': 'KITTY',
                    'BATZ': 'MARU',
                    'MY': 'MELODY'}
        write_json(jsondump, self.file)
        self.assertTrue(os.path.exists(self.file))

        script_path = "%s/helpers/json/dump.py" % self.here
        if VERSION == 2:
            testing_command = ["python2", script_path,
                               '--file', self.file]
        else:
            testing_command = ["python3", script_path,
                               '--file', self.file]

        output = Popen(testing_command,
                       stderr=PIPE,
                       stdout=PIPE)
        t = output.communicate()[0], output.returncode
        result = {'message': t[0],
                  'return_code': t[1]}
        self.assertEqual(result['return_code'], 0)

        output = result['message']
        if isinstance(output, bytes):
            output = output.decode(encoding='UTF-8')

        dump = ['HELLO:"KITTY"', 'BATZ:"MARU"', 'MY:"MELODY"']
        result = output.strip('\n').split('\n')[-3:]
        for res in result:
            self.assertTrue(res in dump)


if __name__ == '__main__':
    unittest.main()
