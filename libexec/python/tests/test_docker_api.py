'''

test_docker_api.py: Docker testing functions for Singularity in Python

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
sys.path.append('..') # directory with docker

from unittest import TestCase
import shutil
import tempfile

VERSION = sys.version_info[0]

print("*** PYTHON VERSION %s API TESTING START ***" %(VERSION))

class TestApi(TestCase):

    def setUp(self):
        self.image = 'docker://ubuntu:latest'
        self.tmpdir = tempfile.mkdtemp()
        os.environ['SINGULARITY_ROOTFS'] = self.tmpdir
        os.mkdir('%s/.singularity.d' %(self.tmpdir))
        from docker.api import DockerApiConnection
        self.client = DockerApiConnection(image=self.image)

        print("\n---START----------------------------------------")

    def tearDown(self):
        shutil.rmtree(self.tmpdir)

        print("---END------------------------------------------")



    def test_get_registry(self):
        '''test_get_registry will verify that the function returns the api base
        with https (default) and verison (default)
        '''
        from docker.api import DockerApiConnection

        print("Case 1: Ask for default registry")
        registry = self.get_registry()
        self.assertEqual(registry,"https://index.docker.io/v2/")

        print("Case 2: Remove https")
        registry = self.get_registry(add_https=False)
        self.assertEqual(registry,"index.docker.io/v2/")

        print("Case 3: Remove version")
        registry = self.get_registry(add_version=False)
        self.assertEqual(registry,"https://index.docker.io")

        print("Case 4: Remove everything")
        registry = self.get_registry(add_version=False,add_https=False)
        self.assertEqual(registry,"index.docker.io")


    def test_get_token(self):
        '''test_get_token will obtain a token from the Docker registry for a namepspace
        and repo. 
        '''
        from docker.api import DockerApiConnection
        client = DockerApiConnection(image="gcr.io/tensorflow/tensorflow:1.0.0")        

        print("Case 1: Ask when we don't need token returns None")
        token = client.update_token()
        self.assertEqual(token,None)


    def test_get_manifest(self):
        '''test_get_manifest will obtain a library/repo manifest
        '''
        from docker.api import DockerApiConnection

        print("Case 1: Obtain manifest for %s/%s" %(self.client.namespace,
                                                    self.client.repo_name))

        manifest = self.client.get_manifest()

        # Default tag should be latest
        self.assertTrue("fsLayers" in manifest or "layers" in manifest)

        # Giving a bad tag sould return error
        print("Case 3: Bad tag should print valid tags and exit")
        client = DockerApiConnection(image="ubuntu:mmm.avocado")        
        
        # Should work for custom registries
        print("Case 4: Obtain manifest from custom registry")
        client = DockerApiConnection(image="gcr.io/tensorflow/tensorflow")        
        manifest = client.get_manifest()
        self.assertTrue("fsLayers" in manifest or "layers" in manifest)


    def test_get_images(self):
        '''test_get_images will obtain a list of images
        '''
        from docker.api import DockerApiConnection

        print("Case 1: Ask for images")
        images = self.client.get_images()
        self.assertTrue(isinstance(images,list))
        self.assertTrue(len(images)>1)

        print("Case 2: Ask for images from custom registry")
        client = DockerApiConnection(image="gcr.io/tensorflow/tensorflow")        
        images = client.get_images()
        self.assertTrue(isinstance(images,list))
        self.assertTrue(len(images)>1)


    def test_get_tags(self):
        '''test_get_tags will obtain a list of tags
        '''
        from docker.api import DockerApiConnection

        print("Case 1: Ask for tags from standard %s/%s" %(self.client.namespace,
                                                           self.client.repo_name))
        tags = self.client.get_tags()
        self.assertTrue(isinstance(tags,list))
        self.assertTrue(len(tags)>1)
        [self.assertTrue(x in tags) for x in ['xenial','latest','trusty','yakkety']]

        print("Case 2: Ask for tags from custom registry")
        client = DockerApiConnection(image="gcr.io/tensorflow/tensorflow")        
        tags = client.get_tags()
        self.assertTrue(isinstance(tags,list))
        self.assertTrue(len(tags)>1)
        [self.assertTrue(x in tags) for x in ['latest','latest-gpu']]
  


    def test_get_layer(self):
        '''test_get_layer will download docker layers
        '''
        from docker.api import DockerApiConnection

        images = self.client.get_images()
        
        print("Case 1: Download an existing layer, should succeed")
        layer_file = self.client.get_layer(image_id=images[0], 
                                           download_folder = self.tmpdir)
        self.assertTrue(os.path.exists(layer_file))


if __name__ == '__main__':
    unittest.main()
