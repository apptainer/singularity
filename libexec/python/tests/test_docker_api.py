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
        self.namespace = 'library'
        self.repo_name = 'ubuntu'
        self.tmpdir = tempfile.mkdtemp()
        os.environ['SINGULARITY_ROOTFS'] = self.tmpdir
        os.mkdir('%s/.singularity' %(self.tmpdir))

        print("\n---START----------------------------------------")

    def tearDown(self):
        shutil.rmtree(self.tmpdir)

        print("---END------------------------------------------")


    def test_create_runscript(self):
        '''test_create_runscript should ensure that a runscript is generated
        with some command
        '''
        print('Testing creation of runscript')
        from docker.api import create_runscript, get_manifest
        from utils import read_file

        manifest = get_manifest(repo_name = self.repo_name,
                                namespace = self.namespace)

        print("Case 1: Asking for CMD when none defined")        
        default_cmd = 'exec /bin/bash "$@"'
        runscript = create_runscript(manifest=manifest,
                                     includecmd=True)
        self.assertTrue(os.path.exists(runscript))
        generated_cmd = read_file(runscript)
        # Commands are always in format exec [] "$@"
        # 'exec echo \'Hello World\' "$@"'
        self.assertTrue(default_cmd in generated_cmd)

        print("Case 2: Asking for ENTRYPOINT when none defined")        
        runscript = create_runscript(manifest=manifest)
        generated_cmd = read_file(runscript)
        self.assertTrue(default_cmd in generated_cmd)

        manifest = get_manifest(repo_name = "mriqc",
                                namespace = "bids",
                                repo_tag="0.0.2")

        print("Case 3: Asking for ENTRYPOINT when defined")        
        runscript = create_runscript(manifest=manifest)
        generated_cmd = read_file(runscript)
        self.assertTrue('exec /usr/bin/run_mriqc "$@"' in generated_cmd)        


        print("Case 4: Asking for CMD when defined")    
          
        runscript = create_runscript(manifest=manifest,
                                     includecmd=True)
        generated_cmd = read_file(runscript)
        self.assertTrue('exec --help "$@"' in generated_cmd)        


        print("Case 5: Asking for ENTRYPOINT when None, should return CMD")    
        from docker.api import get_configs
        manifest = get_manifest(repo_name = "tensorflow",
                                namespace = "tensorflow",
                                repo_tag="1.0.0")
        configs = get_configs(manifest,['Cmd','Entrypoint'])
        self.assertEqual(configs['Entrypoint'],None)
        runscript = create_runscript(manifest=manifest)

        generated_cmd = read_file(runscript)
        self.assertTrue(re.search(configs['Cmd'], ''.join(generated_cmd)))


    def test_get_token(self):
        '''test_get_token will obtain a token from the Docker registry for a namepspace
        and repo. 
        '''
        from docker.api import get_token
        print("Case 1: Ask when we don't need token returns None")
        token = get_token(namespace = 'tensorflow',
                          repo_name = 'tensorflow',
                          registry = 'gcr.io')
        self.assertEqual(token,None)
        print("Case 2: Ask when we do need token returns token header")
        token = get_token(namespace = self.namespace,
                          repo_name = self.repo_name)
        self.assertTrue("Authorization" in token)
        self.assertTrue(len(token["Authorization"])==1437)

        #TODO: test here for different registry auth challenge?


    def test_get_manifest(self):
        '''test_get_manifest will obtain a library/repo manifest
        '''
        from docker.api import get_manifest
        print("Case 1: Obtain manifest for %s/%s" %(self.namespace,
                                                    self.repo_name))

        manifest = get_manifest(repo_name = self.repo_name,
                                namespace = self.namespace)

        # Default tag should be latest
        self.assertEqual(manifest['tag'],"latest")
        self.assertTrue("fsLayers" in manifest)
        self.assertEqual(manifest['name'],"%s/%s" %(self.namespace,
                                                    self.repo_name))

        repo_tag = "14.04"
        print("Case 2: Obtain manifest with custom tag %s" %repo_tag)
        manifest = get_manifest(repo_name = self.repo_name,
                                namespace = self.namespace, 
                                repo_tag = repo_tag)

        # This will trigger if/when they change the schema on us
        self.assertEqual(manifest['schemaVersion'],1)

        # Giving a bad tag sould return error
        print("Case 3: Bad tag should print valid tags and exit")
        with self.assertRaises(SystemExit) as cm:
            manifest = get_manifest(repo_name = self.repo_name,
                                    namespace = self.namespace, 
                                    repo_tag = "mmm.avocado")
        self.assertEqual(cm.exception.code, 1)

        # Should work for custom registries
        print("Case 4: Obtain manifest from custom registry")
        manifest = get_manifest(repo_name = "tensorflow",
                                namespace = "tensorflow", 
                                registry = "gcr.io")
        self.assertTrue("fsLayers" in manifest)


    def test_get_images(self):
        '''test_get_images will obtain a list of images
        '''
        from docker.api import get_images
        from docker.api import get_manifest
        print("Case 1: Ask for images")
        images = get_images(repo_name = self.repo_name,
                            namespace = self.namespace)
        self.assertTrue(isinstance(images,list))
        self.assertTrue(len(images)>1)

        print("Case 2: Ask for images from custom registry")
        images = get_images(repo_name = 'tensorflow',
                            namespace = 'tensorflow',
                            registry = 'gcr.io')
        self.assertTrue(isinstance(images,list))
        self.assertTrue(len(images)>1)


    def test_get_tags(self):
        '''test_get_tags will obtain a list of tags
        '''
        from docker.api import get_tags

        print("Case 1: Ask for tags from standard %s/%s" %(self.namespace,
                                                           self.repo_name))
        tags = get_tags(repo_name = self.repo_name,
                        namespace = self.namespace)
        self.assertTrue(isinstance(tags,list))
        self.assertTrue(len(tags)>1)
        [self.assertTrue(x in tags) for x in ['xenial','latest','trusty','yakkety']]

        print("Case 2: Ask for tags from custom registry")
        tags = get_tags(repo_name = 'tensorflow',
                        namespace = 'tensorflow',
                        registry = 'gcr.io')
        self.assertTrue(isinstance(tags,list))
        self.assertTrue(len(tags)>1)
        [self.assertTrue(x in tags) for x in ['latest','latest-gpu']]
  

    def test_get_config(self):
        '''test_get_config will obtain parameters from the DOcker configuration json
        '''
        from docker.api import get_config
        from docker.api import get_manifest

        # Default should return entrypoint
        print("Case 1: Ask for default command (Entrypoint)")
        manifest = get_manifest(repo_name = self.repo_name,
                                namespace = self.namespace)
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
        from docker.api import get_layer
        from docker.api import get_images
        images = get_images(namespace = self.namespace,
                            repo_name = self.repo_name)
        
        print("Case 1: Download an existing layer, should succeed")
        layer_file = get_layer(image_id=images[0], 
                               repo_name = self.repo_name,
                               namespace = self.namespace,
                               download_folder = self.tmpdir)
        self.assertTrue(os.path.exists(layer_file))

        print("Case 2: Download a non existing layer, should fail")
        fake_layer = "sha256:111111111112222222222223333333333"
        with self.assertRaises(SystemExit) as cm:
            layer_file = get_layer(image_id=fake_layer, 
                                   repo_name = self.repo_name,
                                   namespace = self.namespace,
                                   download_folder = self.tmpdir)
        self.assertEqual(cm.exception.code, 1)


if __name__ == '__main__':
    unittest.main()
