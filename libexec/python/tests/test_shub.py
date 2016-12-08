'''

test_shub.py: Singularity Hub testing functions for Singularity in Python

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
import sys
sys.path.append('..') # directory with singularity, etc.

from unittest import TestCase
from utils import read_file
import shutil
import tempfile

VERSION = sys.version_info[0]

print("*** PYTHON VERSION %s CLIENT TESTING START ***" %(VERSION))

class TestApi(TestCase):


    def setUp(self):
        self.image_id = 8
        self.tmpdir = tempfile.mkdtemp()
        print("\n---START----------------------------------------")

    def tearDown(self):
        shutil.rmtree(self.tmpdir)

        print("---END------------------------------------------")


    def test_get_manifest(self):
        '''test_get_manifest should return the shub manifest
        '''
        from shub.api import get_manifest
        print("Case 1: Testing retrieval of singularity-hub manifest")
        manifest = get_manifest(self.image_id)
        keys = ['files', 'name', 'image', 'collection', 
                'id', 'version', 'spec']
        [self.assertTrue(x in manifest) for x in keys]
        self.assertTrue(manifest['id']==self.image_id)



    def test_download_image(self):
        '''test_download_image will ensure that an image is downloaded to an
        appropriate location (tmpdir) or cache
        '''
        from shub.api import download_image, get_manifest
        print("Case 1: Specifying a directory downloads to it")
        manifest = get_manifest(image_id=self.image_id)
        image = download_image(manifest,
                               download_folder=self.tmpdir)
        self.assertEqual(os.path.dirname(image),self.tmpdir)
        os.remove(image)

        print("Case 2: Not specifying a directory downloads to PWD")
        os.chdir(self.tmpdir)
        manifest = get_manifest(image_id=self.image_id)
        image = download_image(manifest)
        self.assertEqual(os.getcwd(),self.tmpdir)


    def test_get_image_name(self):
        '''test_get_image_name will return the image name from the manifest
        '''
        from shub.api import get_image_name, get_manifest
        manifest = get_manifest(image_id=self.image_id)
        
        print("Case 1: return an image name using the commit id")
        image_name = get_image_name(manifest)
        self.assertEqual('f57e631a0434c31f0b4fa5276a314a6d8a672a55.img.gz',
                         image_name)

        print("Case 2: ask for invalid extension")
        with self.assertRaises(SystemExit) as cm:
            image_name = get_image_name(manifest,
                                        extension='.bz2')
        self.assertEqual(cm.exception.code, 1)

        print("Case 3: don't use commit (use md5 sum on generation)")
        image_name = get_image_name(manifest,
                                    use_commit=False)
        print(image_name)
        self.assertEqual('be4b9ba8fc22525d2ee2b27846513d42.img.gz',image_name)


if __name__ == '__main__':
    unittest.main()
