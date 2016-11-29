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

import re
import sys
sys.path.append('..') # directory with client

from unittest import TestCase
from cli import get_parser, run

class TestClient(TestCase):

    def setUp(self):
        self.parser = get_parser()

    def test_singularity_rootfs(self):
        '''test_singularity_rootfs ensures that --rootfs is required
        '''
        args = self.parser.parse_args([])
        with self.assertRaises(SystemExit) as cm:
            run(args)
        self.assertEqual(cm.exception.code, 1)


class TestUtils(TestCase):

    def setUp(self):
        self.parser = get_parser()

    def test_add_http(self):
        '''test_add_http ensures that http is added to a url
        '''
        from utils import add_http
        url = 'registry.docker.io'

        # Default is https
        http = add_http(url)
        self.assertEqual("https://%s"%url,http)

        # http
        http = add_http(url,use_https=False)
        self.assertEqual("http://%s"%url,http)

        # This should not change. Note - is url is http, stays http
        url = 'https://registry.docker.io'
        http = add_http(url)
        self.assertEqual(url,http)

        #TODO: add test to change http to https


    def test_parse_headers(self):
        '''test_add_http ensures that http is added to a url
        '''
        from utils import parse_headers
        
        # If we don't give headers, and no default, should get {} 
        empty_dict = parse_headers(default_header=False)
        self.assertEqual(empty_dict,dict())

        # If we ask for default, should get something back
        headers = parse_headers(default_header=True)
        for field in ["Accept","Content-Type"]:
            self.assertTrue(field in headers)

        # Can we add a header?
        new_header = {"cookies":"nom"}
        headers = parse_headers(default_header=True,
                                headers=new_header)
        for field in ["Accept","Content-Type","cookies"]:
            self.assertTrue(field in headers)


if __name__ == '__main__':
    unittest.main()
