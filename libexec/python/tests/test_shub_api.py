'''

test_shub.py: Singularity Hub testing functions for Singularity in Python

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
from sutils import read_file
from glob import glob
import shutil
import tempfile

VERSION = sys.version_info[0]

print("*** PYTHON VERSION %s SINGULARITY HUB API TESTING START ***" % VERSION)


class TestApi(TestCase):

    def setUp(self):
        from shub.api import SingularityApiConnection
        self.connect = SingularityApiConnection
        self.image = "shub://vsoch/singularity-images"
        self.client = SingularityApiConnection(image=self.image)

        self.user_name = "vsoch"
        self.repo_name = "singularity-images"
        self.tmpdir = tempfile.mkdtemp()
        os.environ['SINGULARITY_ROOTFS'] = self.tmpdir
        os.mkdir('%s/.singularity.d' % self.tmpdir)

    def tearDown(self):
        shutil.rmtree(self.tmpdir)

    def test_get_manifest(self):
        '''test_get_manifest should return the shub manifest
        '''
        print("Testing singularity hub manifest retrieval")
        print("...case 1: Testing retrieval of singularity-hub manifest")
        manifest = self.client.get_manifest()
        keys = ['version', 'tag', 'branch',
                'name', 'id', 'commit', 'image']
        [self.assertTrue(x in manifest) for x in keys]

    def test_download_image(self):
        '''test_download_image will ensure that
        an image is downloaded to an
        appropriate location (tmpdir) or cache
        '''
        print("Testing singularity hub pull")
        print("...case 1: Specifying a directory downloads to it")
        manifest = self.client.get_manifest()
        image = self.client.download_image(manifest=manifest,
                                           download_folder=self.tmpdir)
        self.assertEqual(os.path.dirname(image), self.tmpdir)

        print("...case 2: Not specifying a directory downloads to PWD")
        os.chdir(self.tmpdir)
        image = self.client.download_image(manifest)
        self.assertEqual(os.getcwd(), self.tmpdir)
        print(image)
        self.assertTrue(image in glob("*"))
        os.remove(image)

    def test_uri(self):
        '''test_uri will make sure that the endpoint returns the equivalent
        image for all different uri options
        '''
        print("Testing singularity hub uris")
        from shub.api import get_image_name
        manifest = self.client.get_manifest()
        image_name = get_image_name(manifest)

        fullname = "%s/%s" % (self.user_name, self.repo_name)
        print("...case 1: ask for image by tag.")
        client = self.connect(image="%s:latest" % fullname)
        manifest = client.get_manifest()
        image_name = get_image_name(manifest)
        self.assertEqual(image_name, get_image_name(manifest))

    def test_get_image_name(self):
        '''test_get_image_name will return the image name from the manifest
        '''
        print("Testing singularity hub image naming")
        from shub.api import get_image_name
        manifest = self.client.get_manifest()
        print("...case 1: return an image name corresponding to repo")
        image_name = get_image_name(manifest)
        self.assertEqual('vsoch-singularity-images-master-latest', image_name)


if __name__ == '__main__':
    unittest.main()
