'''

test_core.py: Python testing for core functions for
              Singularity in Python,
              including defaults, utils, and shell functions.

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
import tarfile
sys.path.append('..')  # noqa

from unittest import TestCase
import shutil
import tempfile

VERSION = sys.version_info[0]

print("*** PYTHON VERSION %s BASE TESTING START ***" % VERSION)


class TestShell(TestCase):

    def setUp(self):

        # Test repo information
        self.registry = "registry"
        self.repo_name = "repo"
        self.namespace = "namespace"
        self.tag = "tag"

        # Default repo information
        self.REGISTRY = 'index.docker.io'
        self.NAMESPACE = 'library'
        self.REPO_TAG = 'latest'
        self.tmpdir = tempfile.mkdtemp()
        os.environ['SINGULARITY_ROOTFS'] = self.tmpdir

        print("\n---START----------------------------------------")

    def tearDown(self):
        shutil.rmtree(self.tmpdir)

        print("---END------------------------------------------")

    def test_get_image_uri(self):
        '''test_get_image_uri ensures that the correct uri is returned
        for a user specified uri, registry, namespace.
        '''
        from shell import get_image_uri
        print("Case 1: No image uri should return None")
        image_uri = get_image_uri('namespace/repo:tag')
        self.assertEqual(image_uri, None)

        print("Case 2: testing return of shub://")
        image_uri = get_image_uri('shub://namespace/repo:tag')
        self.assertEqual(image_uri, 'shub://')

        print("Case 3: testing return of docker uri")
        image_uri = get_image_uri('docker://namespace/repo:tag')
        self.assertEqual(image_uri, 'docker://')

        print("Case 4: weird capitalization should return lowercase")
        image_uri = get_image_uri('DocKer://namespace/repo:tag')
        self.assertEqual(image_uri, 'docker://')

    def test_remove_image_uri(self):
        '''test_remove_image_uri removes the uri
        '''
        from shell import remove_image_uri
        print("Case 1: No image_uri should estimate first")
        image = remove_image_uri('myuri://namespace/repo:tag')
        self.assertEqual(image, "namespace/repo:tag")

        print("Case 2: Missing image uri should return image")
        image = remove_image_uri('namespace/repo:tag')
        self.assertEqual(image, "namespace/repo:tag")

    def test_parse_image_uri(self):
        '''test_parse_image_uri ensures that the correct namespace,
        repo name, and tag (or unique id) is returned.
        '''

        from shell import parse_image_uri

        print("Case 1: Empty repo_name should return error")
        with self.assertRaises(SystemExit) as cm:
            image = parse_image_uri(image="")
        self.assertEqual(cm.exception.code, 1)

        print("Case 2: Checking for correct output tags in digest...")
        image_name = "%s/%s" % (self.namespace, self.repo_name)
        digest = parse_image_uri(image=image_name)
        for tag in ['registry', 'repo_name', 'repo_tag', 'namespace']:
            self.assertTrue(tag in digest)

        print("Case 3: Specifying only an image should return defaults")
        image = parse_image_uri(image="shub://lizardleezle",
                                uri="shub://")
        self.assertTrue(isinstance(image, dict))
        self.assertEqual(image["namespace"], self.NAMESPACE)
        self.assertEqual(image["repo_tag"], self.REPO_TAG)
        self.assertEqual(image["repo_name"], 'lizardleezle')
        self.assertEqual(image["registry"], self.REGISTRY)

        print("Case 4: Tag when speciifed should be returned.")
        image_name = "%s/%s:%s" % (self.namespace,
                                   self.repo_name,
                                   "pusheenasaurus")

        digest = parse_image_uri(image_name)
        self.assertTrue(digest['repo_tag'] == 'pusheenasaurus')

        print("Case 5: Repo name and tag without namespace...")
        image_name = "%s:%s" % (self.repo_name, self.tag)
        digest = parse_image_uri(image_name)
        self.assertTrue(digest['repo_tag'] == self.tag)
        self.assertTrue(digest['namespace'] == 'library')
        self.assertTrue(digest['repo_name'] == self.repo_name)

        print("Case 6: Changing default namespace should not use library.")
        image_name = "meow/%s:%s" % (self.repo_name, self.tag)
        digest = parse_image_uri(image_name)
        self.assertTrue(digest['namespace'] == 'meow')

        print("Case 7: Changing default shouldn't use index.docker.io.")
        image_name = "meow/mix/%s:%s" % (self.repo_name, self.tag)
        digest = parse_image_uri(image_name)
        self.assertTrue(digest['registry'] == 'meow')
        self.assertTrue(digest['namespace'] == 'mix')

        print("Case 8: Custom uri should use it.")
        image_name = "catdog://meow/mix/%s:%s" % (self.repo_name,
                                                  self.tag)
        digest = parse_image_uri(image_name, uri="catdog://")
        self.assertTrue(digest['registry'] == 'meow')
        self.assertTrue(digest['namespace'] == 'mix')

        print("Case 9: Digest version should be parsed")
        image_name = ("catdog://meow/mix/%s:%s@sha:256xxxxxxxxxxxxxxx"
                      % (self.repo_name, self.tag))
        digest = parse_image_uri(image_name, uri="catdog://")
        self.assertTrue(digest['registry'] == 'meow')
        self.assertTrue(digest['namespace'] == 'mix')
        self.assertTrue(digest['version'] == 'sha:256xxxxxxxxxxxxxxx')


class TestUtils(TestCase):

    def setUp(self):
        self.tmpdir = tempfile.mkdtemp()
        print("\n---START----------------------------------------")

    def tearDown(self):
        shutil.rmtree(self.tmpdir)
        print("---END------------------------------------------")

    def test_add_http(self):
        '''test_add_http ensures that http is added to a url
        '''

        from sutils import add_http
        url_http = 'http://registry.docker.io'
        url_https = 'https://registry.docker.io'

        print("Case 1: adding https to url with nothing specified...")

        # Default is https
        url = 'registry.docker.io'
        http = add_http(url)
        self.assertEqual(url_https, http)

        # http
        print("Case 2: adding http to url with nothing specified...")
        http = add_http(url, use_https=False)
        self.assertEqual(url_http, http)

        # This should not change. Note - is url is http, stays http
        print("Case 3: url already has https, should not change...")
        url = 'https://registry.docker.io'
        http = add_http(url)
        self.assertEqual(url_https, http)

        # This should not change. Note - is url is http, stays http
        print("Case 4: url already has http, should not change...")
        url = 'http://registry.docker.io'
        http = add_http(url, use_https=False)
        self.assertEqual(url_http, http)

        print("Case 5: url has http, should change to https")
        url = 'http://registry.docker.io'
        http = add_http(url)
        self.assertEqual(url_https, http)

        print("Case 6: url has https, should change to http")
        url = 'https://registry.docker.io'
        http = add_http(url, use_https=False)
        self.assertEqual(url_http, http)

        print("Case 7: url should have trailing slash stripped")
        url = 'https://registry.docker.io/'
        http = add_http(url, use_https=False)
        self.assertEqual(url_http, http)

    def test_headers(self):
        '''test_add_http ensures that http is added to a url
        '''
        print("Testing utils header functions...")

        from sutils import basic_auth_header

        # Basic auth header
        print("Case 4: ask for custom authentication header")
        auth = basic_auth_header(username='vanessa',
                                 password='pancakes')
        self.assertEqual(auth['Authorization'],
                         'Basic dmFuZXNzYTpwYW5jYWtlcw==')

    def test_run_command(self):
        '''test_run_command tests sending a command to commandline
        using subprocess
        '''
        print("Testing utils.run_command...")

        from sutils import run_command

        # An error should return None
        print("Case 1: Command errors returns None ")
        none = run_command(['exec', 'whaaczasd'])
        self.assertEqual(none, None)

        # A success should return console output
        print("Case 2: Command success returns output")
        hello = run_command(['echo', 'hello'])
        if not isinstance(hello, str):  # python 3 support
            hello = hello.decode('utf-8')
        self.assertEqual(hello, 'hello\n')

    def test_is_number(self):
        '''test_is_number should return True for any string or
        number that turns to a number, and False for everything else
        '''
        print("Testing utils.is_number...")

        from sutils import is_number

        print("Case 1: Testing string and float numbers returns True")
        self.assertTrue(is_number("4"))
        self.assertTrue(is_number(4))
        self.assertTrue(is_number("2.0"))
        self.assertTrue(is_number(2.0))

        print("Case 2: Testing repo names, tags, commits, returns False")
        self.assertFalse(is_number("vsoch/singularity-images"))
        self.assertFalse(is_number("vsoch/singularity-images:latest"))
        self.assertFalse(is_number("44ca6e7c6c35778ab80b34c3fc940c32f1810f39"))

    def test_extract_tar(self):
        '''test_extract_tar will test extraction of a tar.gz file
        '''
        print("Testing utils.extract_tar...")

        # First create a temporary tar file
        from sutils import extract_tar
        from glob import glob
        import tarfile

        # Create and close a temporary tar.gz
        print("Case 1: Testing tar.gz...")
        creation_dir = tempfile.mkdtemp()
        archive, files = create_test_tar(creation_dir)

        # Extract to different directory
        extract_dir = tempfile.mkdtemp()
        extract_tar(archive=archive,
                    output_folder=extract_dir)
        extracted_files = [x.replace(extract_dir, '')
                           for x in glob("%s/tmp/*" % extract_dir)]
        [self.assertTrue(x in files) for x in extracted_files]

        # Clean up
        for dirname in [extract_dir, creation_dir]:
            shutil.rmtree(dirname)

        print("Case 2: Testing tar...")
        creation_dir = tempfile.mkdtemp()
        archive, files = create_test_tar(creation_dir,
                                         compressed=False)

        # Extract to different directory
        extract_dir = tempfile.mkdtemp()
        extract_tar(archive=archive,
                    output_folder=extract_dir)
        extracted_files = [x.replace(extract_dir, '')
                           for x in glob("%s/tmp/*" % extract_dir)]
        [self.assertTrue(x in files) for x in extracted_files]

        print("Case 3: Testing that extract_tar returns None on error...")
        creation_dir = tempfile.mkdtemp()
        archive, files = create_test_tar(creation_dir,
                                         compressed=False)
        extract_dir = tempfile.mkdtemp()
        shutil.rmtree(extract_dir)
        output = extract_tar(archive=archive,
                             output_folder=extract_dir)
        self.assertEqual(output, None)

    def test_write_read_files(self):
        '''test_write_read_files will test the
        functions write_file and read_file
        '''
        print("Testing utils.write_file...")
        from sutils import write_file
        import json
        tmpfile = tempfile.mkstemp()[1]
        os.remove(tmpfile)
        write_file(tmpfile, "hello!")
        self.assertTrue(os.path.exists(tmpfile))

        print("Testing utils.read_file...")
        from sutils import read_file
        content = read_file(tmpfile)[0]
        self.assertEqual("hello!", content)

        from sutils import write_json
        print("Testing utils.write_json...")
        print("Case 1: Providing bad json")
        bad_json = {"Wakkawakkawakka'}": [{True}, "2", 3]}
        tmpfile = tempfile.mkstemp()[1]
        os.remove(tmpfile)
        with self.assertRaises(TypeError) as cm:
            write_json(bad_json, tmpfile)

        print("Case 2: Providing good json")
        good_json = {"Wakkawakkawakka": [True, "2", 3]}
        tmpfile = tempfile.mkstemp()[1]
        os.remove(tmpfile)
        write_json(good_json, tmpfile)
        content = json.load(open(tmpfile, 'r'))
        self.assertTrue(isinstance(content, dict))
        self.assertTrue("Wakkawakkawakka" in content)

    def test_clean_path(self):
        '''test_clean_path will test the clean_path function
        '''
        print("Testing utils.clean_path...")
        from sutils import clean_path
        ideal_path = '/home/vanessa/stuff'
        self.assertEqual(clean_path('/home/vanessa/stuff/'), ideal_path)
        self.assertEqual(clean_path('/home/vanessa/stuff//'), ideal_path)
        self.assertEqual(clean_path('/home/vanessa//stuff/'), ideal_path)

    def test_get_fullpath(self):
        '''test_get_fullpath will test the get_fullpath function
        '''
        print("Testing utils.get_fullpath...")
        from sutils import get_fullpath
        tmpfile = tempfile.mkstemp()[1]

        print("Case 1: File exists, should return full path")
        self.assertEqual(get_fullpath(tmpfile), tmpfile)

        print("Case 2: File doesn't exist, should return error")
        os.remove(tmpfile)
        with self.assertRaises(SystemExit) as cm:
            get_fullpath(tmpfile)
        self.assertEqual(cm.exception.code, 1)

        print("Case 3: File doesn't exist, should return None")
        self.assertEqual(get_fullpath(tmpfile,
                                      required=False), None)

    def test_write_singularity_infos(self):
        '''test_get_fullpath will test the get_fullpath function
        '''
        print("Testing utils.write_singuarity_infos...")
        from sutils import write_singularity_infos
        base_dir = '%s/ROOTFS' % self.tmpdir
        prefix = 'docker'
        start_number = 0
        content = "export HELLO=MOTO"

        print("Case 1: Metadata base doesn't exist, should return error")
        with self.assertRaises(SystemExit) as cm:
            info_file = write_singularity_infos(base_dir=base_dir,
                                                prefix=prefix,
                                                start_number=start_number,
                                                content=content)
        self.assertEqual(cm.exception.code, 1)

        print("Case 2: Metadata base does exist, should return path.")
        os.mkdir(base_dir)
        info_file = write_singularity_infos(base_dir=base_dir,
                                            prefix=prefix,
                                            start_number=start_number,
                                            content=content)
        self.assertEqual(info_file, "%s/%s-%s" % (base_dir,
                                                  start_number,
                                                  prefix))

        print("Case 3: Adding another equivalent prefix should return next")
        info_file = write_singularity_infos(base_dir=base_dir,
                                            prefix=prefix,
                                            start_number=start_number,
                                            content=content)
        self.assertEqual(info_file, "%s/%s-%s" % (base_dir,
                                                  start_number+1,
                                                  prefix))

        print("Case 4: Files have correct content.")
        with open(info_file, 'r') as filey:
            written_content = filey.read()
        self.assertEqual(content, written_content)


# Supporting Test Functions
def create_test_tar(tmpdir, compressed=True):
    archive = "%s/toodles.tar.gz" % tmpdir
    if compressed is False:
        archive = "%s/toodles.tar" % tmpdir
    mode = "w:gz"
    if compressed is False:
        mode = "w"
    print("Creating %s" % archive)
    tar = tarfile.open(archive, mode)
    files = [tempfile.mkstemp()[1] for x in range(3)]
    [tar.add(x) for x in files]
    tar.close()
    return archive, files


if __name__ == '__main__':
    unittest.main()
