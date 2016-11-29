'''

test_cli.py: Client (cli.py) testing functions for Singularity in Python

Copyright (c) 2016, Vanessa Sochat. All rights reserved. 

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
sys.path.append('..') # directory with client

from unittest import TestCase
from cli import get_parser, run
import shutil
import tempfile

VERSION = sys.version_info[0]

print("*** PYTHON VERSION %s CLIENT TESTING START ***" %(VERSION))

class TestClient(TestCase):

    def setUp(self):
        print("\n---START----------------------------------------")
        self.parser = get_parser()

    def tearDown(self):
        print("---END------------------------------------------")

    def test_singularity_rootfs(self):
        '''test_singularity_rootfs ensures that --rootfs is required
        '''
        print("Testing --rootfs command...")
        args = self.parser.parse_args([])
        with self.assertRaises(SystemExit) as cm:
            run(args)
        self.assertEqual(cm.exception.code, 1)


class TestUtils(TestCase):

    def setUp(self):
        print("\n---START----------------------------------------")

    def tearDown(self):
        print("---END------------------------------------------")

    def test_add_http(self):
        '''test_add_http ensures that http is added to a url
        '''
        print("Case 1: adding https to url with nothing specified...")

        from utils import add_http
        url = 'registry.docker.io'

        # Default is https
        http = add_http(url)
        self.assertEqual("https://%s"%url,http)

        # http
        print("Case 2: adding http to url with nothing specified...")
        http = add_http(url,use_https=False)
        self.assertEqual("http://%s"%url,http)

        # This should not change. Note - is url is http, stays http
        print("Case 3: url already has https, should not change...")
        url = 'https://registry.docker.io'
        http = add_http(url)
        self.assertEqual(url,http)

        #TODO: add test to change http to https


    def test_headers(self):
        '''test_add_http ensures that http is added to a url
        '''
        print("Testing utils header functions...")

        from utils import parse_headers
        from utils import basic_auth_header
        
        # If we don't give headers, and no default, should get {} 
        print("Case 1: Don't give headers, return empty dictionary")
        empty_dict = parse_headers(default_header=False)
        self.assertEqual(empty_dict,dict())

        # If we ask for default, should get something back
        print("Case 2: ask for default headers...")
        headers = parse_headers(default_header=True)
        for field in ["Accept","Content-Type"]:
            self.assertTrue(field in headers)

        # Can we add a header?
        print("Case 3: add a custom header")
        new_header = {"cookies":"nom"}
        headers = parse_headers(default_header=True,
                                headers=new_header)
        for field in ["Accept","Content-Type","cookies"]:
            self.assertTrue(field in headers)


        # Basic auth header
        print("Case 4: basic_auth_header - ask for custom authentication header")
        auth = basic_auth_header(username='vanessa',
                                 password='pancakes')
        self.assertEqual(auth['Authorization'],
                         'Basic dmFuZXNzYTpwYW5jYWtlcw==')


    def test_run_command(self):
        '''test_run_command tests sending a command to commandline
        using subprocess
        '''
        print("Testing utils.run_command...")

        from utils import run_command
        
        # An error should return None
        print("Case 1: Command errors returns None ")
        none  = run_command(['exec','whaaczasd'])
        self.assertEqual(none,None)

        # A success should return console output
        print("Case 2: Command success returns output")
        hello  = run_command(['echo','hello'])
        if not isinstance(hello,str): # python 3 support
            hello = hello.decode('utf-8')
        self.assertEqual(hello,'hello\n')


    def test_get_cache(self):
        '''test_run_command tests sending a command to commandline
        using subprocess
        '''
        print("Testing utils.get_cache...")

        from utils import get_cache
        
        # If there is no cache_base, we should get default
        print("Case 1: No cache base returns default")
        home = os.environ['HOME']
        cache = get_cache()
        self.assertEqual("%s/.singularity" %home,cache)
        self.assertTrue(os.path.exists(cache))

        # If we give a base, we should get that base instead
        print("Case 2: custom specification of cache base")
        cache_base = '%s/cache' %(home)
        cache = get_cache(cache_base=cache_base)
        self.assertEqual(cache_base,cache)
        self.assertTrue(os.path.exists(cache))

        # If we specify a subfolder, we should get that added
        print("Case 3: Ask for subfolder in cache base")
        subfolder = 'docker'
        cache = get_cache(subfolder=subfolder)
        self.assertEqual("%s/.singularity/%s" %(home,subfolder),cache)
        self.assertTrue(os.path.exists(cache))

        # If we disable the cache, we should get temporary directory
        print("Case 4: Disable the cache (uses /tmp)")
        cache = get_cache(disable_cache=True)
        self.assertTrue(os.path.exists(cache))
        self.assertTrue(re.search("tmp",cache)!=None)

        # If environmental variable set, should use that
        print("Case 5: cache base obtained from environment")
        SINGULARITY_CACHEDIR = '%s/cache' %(home)
        os.environ['SINGULARITY_CACHEDIR'] = SINGULARITY_CACHEDIR
        cache = get_cache()
        self.assertEqual(SINGULARITY_CACHEDIR,cache)
        self.assertTrue(os.path.exists(cache))


    def test_change_permission(self):
        '''test_change_permissions will make sure that we can change
        permissions of a file
        '''
        print("Testing utils.change_permissions...")

        from utils import change_permissions
        from stat import ST_MODE
        
        tmpdir = tempfile.mkdtemp()
        tmpfile = '%s/.mooza' %(tmpdir)
        os.system('touch %s' %(tmpfile))

        # 664
        permissions = oct(os.stat(tmpfile)[ST_MODE])[-3:]
        self.assertTrue(permissions,'664')
        # to 755
        change_permissions(tmpfile,permission="0755")  
        new_permissions = oct(os.stat(tmpfile)[ST_MODE])[-3:]
        self.assertTrue(new_permissions,'755')
        # and back
        change_permissions(tmpfile,permission="0644")  
        new_permissions = oct(os.stat(tmpfile)[ST_MODE])[-3:]
        self.assertTrue(new_permissions,'664')

        shutil.rmtree(tmpdir)


    #TODO: need to test api_get
        

if __name__ == '__main__':
    unittest.main()
