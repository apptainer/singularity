// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package image

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func createVirtualBlockDevice(t *testing.T, path string) {
	cmdBin := "/bin/dd"
	if _, statErr := os.Stat(cmdBin); os.IsNotExist(statErr) {
		t.Skipf("%s not available, skipping the test", cmdBin)
	}

	arg := "of=" + path
	cmd := exec.Command(cmdBin, "if=/dev/zero", arg, "bs=1024", "count=10000")
	cmdErr := cmd.Run()
	if cmdErr != nil {
		os.RemoveAll(path)
		t.Fatalf("cannot create virtual block device: %s", cmdErr)
	}
}

func createFS(t *testing.T, fsType string, path string) {
	cmdBin := "/sbin/mkfs." + fsType
	if _, statErr := os.Stat(cmdBin); os.IsNotExist(statErr) {
		t.Skipf("%s not available, skipping the test", cmdBin)
	}

	cmd := exec.Command(cmdBin, path)
	cmdErr := cmd.Run()
	if cmdErr != nil {
		t.Fatalf("command failed: %s", cmdErr)
	}
}

// Wanring the file will be initialized!!!!
func createFullVirtualBlockDevice(t *testing.T, path string, fsType string) {
	createVirtualBlockDevice(t, path)
	createFS(t, fsType, path)
}

func initializerTest(t *testing.T, img *Image, path string, fsType string) error {
	createFullVirtualBlockDevice(t, path, fsType)

	var err error
	img.File, err = os.Open(path)
	if err != nil {
		t.Fatalf("cannot open file: %s", err)
	}

	fileinfo, err := img.File.Stat()
	if err != nil {
		t.Fatalf("cannot stat image: %s", err)
	}

	var ext3format ext3Format
	err = ext3format.initializer(img, fileinfo)
	// err is just to be returned and analyzed by the caller

	img.File.Close()

	return err
}

func TestCheckExt3Header(t *testing.T) {
	b := make([]byte, bufferSize)

	// Create a fake ext3 file
	dir, err := ioutil.TempDir("", "ext3testing-")
	if err != nil {
		t.Fatalf("impossible to create temporary directory: %s", err)
	}

	path := dir + "ext.fs"

	createFullVirtualBlockDevice(t, path, "ext3")
	defer os.Remove(path)

	// Now load the image
	img, imgErr := os.Open(path)
	if imgErr != nil {
		t.Fatal("impossible to load image for testing")
	}
	defer img.Close()

	n, err := img.Read(b)
	if err != nil || n != bufferSize {
		t.Fatalf("cannot read the first %d bytes of the image", bufferSize)
	}

	_, checkErr := CheckExt3Header(b)
	if checkErr != nil {
		t.Fatalf("cannot check ext3 header of a valid image (%s): %s", path, checkErr)
	}
}

func TestInitializer(t *testing.T) {
	// Create a temporary image which is obviously an invalid ext3 image
	f, err := ioutil.TempFile("", "image-")
	if err != nil {
		t.Fatalf("cannot create temporary file: %s", err)
	}
	path, err := filepath.Abs(f.Name())
	if err != nil {
		f.Close()
		t.Fatalf("impossible to retrieve path of temporary file: %s", err)
	}
	defer os.Remove(path)
	// We do not use defer f.Close() since we will be manually
	// opening and closing the file for testing.
	f.Close()
	resolvedPath, err := ResolvePath(path)
	if err != nil {
		t.Fatalf("failed to retrieve path for %s: %s", path, err)
	}

	img := &Image{
		Path: resolvedPath,
		Name: "test",
	}

	img.Writable = true
	var ext3format ext3Format
	mode := ext3format.openMode(true)
	if mode != os.O_RDWR {
		t.Fatalf("wrong mode returned")
	}
	img.File, err = os.OpenFile(resolvedPath, mode, 0)
	if err != nil {
		t.Fatalf("cannot open the image: %s", err)
	}
	defer img.File.Close()
	fileinfo, err := img.File.Stat()
	if err != nil {
		t.Fatalf("cannot stat image: %s", err)
	}

	// This test will fail because we did not set a valid ext3 FS yet
	err = ext3format.initializer(img, fileinfo)
	if err == nil {
		t.Fatalf("initializer succeeded while expected to fail")
	}

	// Now we setup a valid ext3 FS and run a test again
	err = initializerTest(t, img, resolvedPath, "ext3")
	if err != nil {
		t.Fatalf("ext3 initializer test failed with a valid ext3 image: %s", err)
	}

	// We now run a test with ext2 to hit some other corner cases
	err = initializerTest(t, img, resolvedPath, "ext2")
	if err == nil {
		t.Fatal("ext3 initializer test succeeded with an ext2 image while expected to fail")
	}

	// We reformat the image with different file systems and see if we catch
	// the error
	_, statErr := os.Stat("/sbin/mkfs.fat")
	if statErr == nil {
		err = initializerTest(t, img, resolvedPath, "vfat")
		if err == nil {
			t.Fatalf("ext3 initializer test succeeded with a vfat image")
		}
	} else {
		t.Log("/sbin/mkfs.fat is not available, skipping the test...")
	}

	_, statErr = os.Stat("/sbin/mkfs.ext4")
	if statErr == nil {
		err = initializerTest(t, img, resolvedPath, "ext4")
		if err == nil {
			t.Fatalf("ext3 initializer test succeeded with a ext4 image while expected to fail")
		}
	} else {
		t.Log("/sbin/mkfs.ext4 is not available, skipping the test...")
	}

	// A small test to exercise openMode() when using read-only mode
	mode = ext3format.openMode(false)
	if mode != os.O_RDONLY {
		t.Fatalf("wrong mode returned")
	}

	// Erro case when a directory is passed in to initializer()
	path, err = ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("Cannot create a temporary directory: %s", err)
	}
	defer os.RemoveAll(path)
	resolvedPath, err = ResolvePath(path)
	if err != nil {
		t.Fatalf("failed to retrieve path for %s: %s", path, err)
	}
	img.Path = path
	img.File, err = os.Open(path)
	if err != nil {
		t.Fatalf("cannot open %s: %s", path, err)
	}
	fileinfo, err = img.File.Stat()
	if err != nil {
		t.Fatalf("cannot stat image: %s", err)
	}
	if fileinfo.IsDir() == false {
		t.Fatalf("invalid fileinfo for %s", path)
	}

	err = ext3format.initializer(img, fileinfo)
	if err == nil {
		t.Fatalf("ext3 initializer succeeded with a directory while expected to fail")
	}
}
