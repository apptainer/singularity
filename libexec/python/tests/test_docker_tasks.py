'''

test_docker_tasks.py: Docker tasks testing for Singularity in Python

Copyright (c) 2017, Vanessa Sochat. All rights reserved.

'''

import os
import re
import sys
sys.path.append('..')  # noqa

from unittest import TestCase
import shutil
import tempfile

VERSION = sys.version_info[0]

print("*** PYTHON VERSION %s TASKS TESTING START ***" % VERSION)


class TestApi(TestCase):

    def setUp(self):
        self.image = 'docker://ubuntu:latest'
        self.tmpdir = tempfile.mkdtemp()
        os.environ['SINGULARITY_ROOTFS'] = self.tmpdir
        os.mkdir('%s/.singularity.d' % self.tmpdir)
        from docker.api import DockerApiConnection
        self.client = DockerApiConnection(image=self.image)

        print("\n---START----------------------------------------")

    def tearDown(self):
        shutil.rmtree(self.tmpdir)

        print("---END------------------------------------------")

    def test_create_runscript(self):
        '''test_create_runscript should ensure that
        a runscript is generated with some command
        '''
        from docker.api import DockerApiConnection

        print('Testing creation of runscript')
        from docker.tasks import extract_runscript

        manifest = self.client.get_manifest(old_version=True)

        print("Case 1: Asking for CMD when none defined")
        default_cmd = 'exec "/bin/bash"'
        runscript = extract_runscript(manifest=manifest,
                                      includecmd=True)
        self.assertTrue(default_cmd in runscript)

        print("Case 2: Asking for ENTRYPOINT when none defined")
        runscript = extract_runscript(manifest=manifest)
        self.assertTrue(default_cmd in runscript)

        client = DockerApiConnection(image="docker://bids/mriqc:0.0.2")
        manifest = client.get_manifest(old_version=True)

        print("Case 3: Asking for ENTRYPOINT when defined")
        runscript = extract_runscript(manifest=manifest)
        self.assertTrue('exec "/run_mriqc"' in runscript)

        print("Case 4: Asking for CMD when defined")
        runscript = extract_runscript(manifest=manifest,
                                      includecmd=True)
        self.assertTrue('exec "--help"' in runscript)

        print("Case 5: Asking for ENTRYPOINT when None, should return CMD")
        from docker.tasks import get_configs
        client = DockerApiConnection(image="tensorflow/tensorflow:1.0.0")
        manifest = client.get_manifest(old_version=True)

        configs = get_configs(manifest, ['Cmd', 'Entrypoint'])
        self.assertEqual(configs['Entrypoint'], None)
        runscript = extract_runscript(manifest=manifest)
        self.assertTrue(configs['Cmd'][0] in runscript)

    def test_get_config(self):
        '''test_get_config will obtain parameters
        from the DOcker configuration json
        '''
        from docker.api import DockerApiConnection
        from docker.tasks import get_config

        # Default should return entrypoint
        print("Case 1: Ask for default command (Entrypoint)")
        manifest = self.client.get_manifest(old_version=True)
        entrypoint = get_config(manifest=manifest)

        # Ubuntu latest should have None
        self.assertEqual(entrypoint, None)

        print("Case 2: Ask for custom command (Cmd)")
        entrypoint = get_config(manifest=manifest,
                                spec="Cmd")
        self.assertTrue('/bin/bash' in entrypoint)


if __name__ == '__main__':
    unittest.main()
