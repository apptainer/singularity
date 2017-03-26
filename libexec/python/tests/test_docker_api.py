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
        os.mkdir('%s/singularity.d' %(self.tmpdir))
        from docker.api import DockerApiConnection
        self.client = DockerApiConnection(image=self.image)

        print("\n---START----------------------------------------")

    def tearDown(self):
        shutil.rmtree(self.tmpdir)

        print("---END------------------------------------------")


    def test_create_runscript(self):
        '''test_create_runscript should ensure that a runscript is generated
        with some command
        '''
        from docker.api import DockerApiConnection

        print('Testing creation of runscript')
        from docker.api import extract_runscript

        manifest = self.client.get_manifest(old_version=True)

        print("Case 1: Asking for CMD when none defined")        
        default_cmd = 'exec /bin/bash "$@"'
        runscript = extract_runscript(manifest=manifest,
                                     includecmd=True)
        # Commands are always in format exec [] "$@"
        # 'exec echo \'Hello World\' "$@"'
        self.assertTrue(re.search(default_cmd,runscript) is not None)

        print("Case 2: Asking for ENTRYPOINT when none defined")        
        runscript = extract_runscript(manifest=manifest)
        self.assertTrue(default_cmd in runscript.split('\n'))

        client = DockerApiConnection(image="docker://bids/mriqc:0.0.2")        
        manifest = client.get_manifest(old_version=True)

        print("Case 3: Asking for ENTRYPOINT when defined")        
        runscript = extract_runscript(manifest=manifest)
        self.assertTrue('exec /usr/bin/run_mriqc "$@"' in runscript.split('\n'))        

        print("Case 4: Asking for CMD when defined")              
        runscript = extract_runscript(manifest=manifest,
                                      includecmd=True)
        self.assertTrue('exec --help "$@"' in runscript.split('\n'))        

        print("Case 5: Asking for ENTRYPOINT when None, should return CMD")    
        from docker.api import get_configs
        client = DockerApiConnection(image="tensorflow/tensorflow:1.0.0")        
        manifest = client.get_manifest(old_version=True)

        configs = get_configs(manifest,['Cmd','Entrypoint'])
        self.assertEqual(configs['Entrypoint'],None)
        runscript = extract_runscript(manifest=manifest)
        self.assertTrue(re.search(configs['Cmd'], runscript))


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
        
        with self.assertRaises(SystemExit) as cm:
            manifest = client.get_manifest()
        self.assertEqual(cm.exception.code, 1)

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
  

    def test_get_config(self):
        '''test_get_config will obtain parameters from the DOcker configuration json
        '''
        from docker.api import DockerApiConnection

        from docker.api import get_config

        # Default should return entrypoint
        print("Case 1: Ask for default command (Entrypoint)")
        manifest = self.client.get_manifest(old_version=True)
        entrypoint = get_config(manifest=manifest)

        # Ubuntu latest should have None
        self.assertEqual(entrypoint,None)
        
        print("Case 2: Ask for custom command (Cmd)")
        entrypoint = get_config(manifest=manifest,
                                spec="Cmd")
        self.assertEqual(entrypoint,'/bin/bash')



    def test_get_layer(self):
        '''test_get_layer will download docker layers
        '''
        from docker.api import DockerApiConnection

        images = self.client.get_images()
        
        print("Case 1: Download an existing layer, should succeed")
        layer_file = self.client.get_layer(image_id=images[0], 
                                           download_folder = self.tmpdir)
        self.assertTrue(os.path.exists(layer_file))

        print("Case 2: Download a non existing layer, should fail")
        fake_layer = "sha256:111111111112222222222223333333333"
        with self.assertRaises(SystemExit) as cm:
            layer_file = self.client.get_layer(image_id=fake_layer, 
                                               download_folder = self.tmpdir)
        self.assertEqual(cm.exception.code, 1)


if __name__ == '__main__':
    unittest.main()
