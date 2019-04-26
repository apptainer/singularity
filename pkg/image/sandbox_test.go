// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package image

import (
	"io/ioutil"
	"os"
	"testing"
)

// runSandboxInitializerTest does all the required calls to be able to invoke
// the sandbox initializer, i.e., open the path associated to the image, get
// stats for the associated file/directory and finailly invoke the initializer.
// Note that the function only returns an error in order to let the caller
// decide if it is actually an error for the test or not. This can basically be
// called for both valid and invalid test cases.
// @param[in] test handle so we can stop the test when facing an error
// @param[in] image handle, here we just need the structure, not an actual fully functional image
// @param[in] path to a file or directory that represents a virtual sandbox (a directory being a valid image; a file an invalid one)
// @return error handle that represents the result of the initializer execution. This handle is returned to the caller as only the caller can know if it is a valid/invalid test case.
func runSandboxInitializerTest(t *testing.T, img *Image, path string) error {
	var sandboxfmt sandboxFormat
	var err error
	img.Path = path
	img.File, err = os.Open(path)
	if err != nil {
		t.Fatalf("cannot open file: %s\n", err)
	}

	fileinfo, statErr := img.File.Stat()
	if statErr != nil {
		t.Fatalf("cannot stat file %s: %s\n", path, statErr)
	}

	err = sandboxfmt.initializer(img, fileinfo)
	// Only the caller can interpret the result (valid vs. invalid test case)
	return err
}

func TestSandboxInitializer(t *testing.T) {
	// Valid case using a directory
	path, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("cannot create a temporary directory: %s\n", err)
	}
	defer os.RemoveAll(path)

	img := &Image{
		Path: path,
		Name: "test",
	}

	err = runSandboxInitializerTest(t, img, path)
	if err != nil {
		t.Fatalf("sandbox initializer failed: %s\n", err)
	}

	// Invalid case using a file
	f, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("cannot create temporary file: %s\n", err)
	}
	defer f.Close()

	path = f.Name()
	defer os.Remove(path)

	f.Close()

	err = runSandboxInitializerTest(t, img, path)
	if err == nil {
		t.Fatal("sandbox initializer using a file succeeded while expected to fail")
	}
}

func TestSBOpenMode(t *testing.T) {
	var sandboxfmt sandboxFormat

	// Yes, it is correct, openMode() always return 'os.O_RDONLY'
	if sandboxfmt.openMode(true) != os.O_RDONLY {
		t.Fatal("openMode(true) returned the wrong value")
	}

	if sandboxfmt.openMode(false) != os.O_RDONLY {
		t.Fatal("openMode(false) returned the wrong value")
	}
}
