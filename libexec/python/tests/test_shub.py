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
from glob import glob
import shutil
import tempfile

VERSION = sys.version_info[0]

print("*** PYTHON VERSION %s CLIENT TESTING START ***" %(VERSION))

class TestApi(TestCase):


    def setUp(self):
        self.image_id = 60 # https://singularity-hub.org/collections/12/
        self.user_name = "vsoch"
        self.repo_name = "singularity-images"
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
        keys = ['files', 'version', 'collection', 'branch', 
                'name', 'id', 'metrics', 'spec', 'image']
        [self.assertTrue(x in manifest) for x in keys]
        self.assertTrue(manifest['id']==self.image_id)



    def test_download_image(self):
        '''test_download_image will ensure that an image is downloaded to an
        appropriate location (tmpdir) or cache
        '''
        from shub.api import download_image, get_manifest
        print("Case 1: Specifying a directory downloads to it")
        manifest = get_manifest(image=self.image_id)
        image = download_image(manifest,
                               download_folder=self.tmpdir)
        self.assertEqual(os.path.dirname(image),self.tmpdir)
        
        print("Case 2: Image should be named based on commit.")
        image_name = os.path.splitext(os.path.basename(image))[0]
        self.assertEqual(image_name,manifest['version'])
        os.remove(image)

        print("Case 3: Not specifying a directory downloads to PWD")
        os.chdir(self.tmpdir)
        image = download_image(manifest)
        self.assertEqual(os.getcwd(),self.tmpdir)
        self.assertTrue(image in glob("*"))
        os.remove(image)

        print("Case 4: Image should not be extracted.")
        image = download_image(manifest,extract=False)
        self.assertTrue(image.endswith('.img.gz'))        

    def test_uri(self):
        '''test_uri will make sure that the endpoint returns the equivalent
        image for all different uri options
        '''
        from shub.api import get_image_name, get_manifest
        manifest = get_manifest(image=self.image_id)
        image_name = get_image_name(manifest)
                
        print("Case 1: ask for image and ask for master branch (tag)")
        manifest = get_manifest(image="%s/%s:master" %(self.user_name,self.repo_name))
        self.assertEqual(image_name,get_image_name(manifest))

        print("Case 2: ask for different tag (mongo)")
        manifest = get_manifest(image="%s/%s:mongo" %(self.user_name,self.repo_name))
        mongo = get_image_name(manifest)
        self.assertFalse(image_name == mongo)

        print("Case 3: ask for image without tag (should be latest across tags, mongo)")
        manifest = get_manifest(image="%s/%s" %(self.user_name,self.repo_name))
        self.assertEqual(mongo,get_image_name(manifest))

        print("Case 4: ask for latest tag (should be latest across tags, mongo)")
        manifest = get_manifest(image="%s/%s:latest" %(self.user_name,self.repo_name))
        self.assertEqual(mongo,get_image_name(manifest))


    def test_get_image_name(self):
        '''test_get_image_name will return the image name from the manifest
        '''
        from shub.api import get_image_name, get_manifest
        manifest = get_manifest(image=self.image_id)
                
        print("Case 1: return an image name using the commit id")
        image_name = get_image_name(manifest)
        self.assertEqual('6d3715a982865863ff20e8783014522edf1240e4.img.gz',
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
        self.assertEqual('9e46ba8be1e10b1a2812844ac8072259.img.gz',image_name)


if __name__ == '__main__':
    unittest.main()
