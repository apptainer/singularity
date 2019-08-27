// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package image

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
)

// createVirtualBlockDevice creates a virtual block device
// in a file using the dd command. The test is skipped if
// the command is not available.
// @parma[in] test handle to control the test, i.e., stop it in case of fatal error
// @parma[in] path to the virtual block device to be created
func createVirtualBlockDevice(t *testing.T, path string) {
	cmdBin, err := exec.LookPath("dd")
	if err != nil {
		t.Skip("dd command not available, skipping the test")
	}

	arg := "of=" + path
	cmd := exec.Command(cmdBin, "if=/dev/zero", arg, "bs=1024", "count=10000")
	cmdErr := cmd.Run()
	if cmdErr != nil {
		os.RemoveAll(path)
		t.Fatalf("cannot create virtual block device: %s\n", cmdErr)
	}
}

// Create a new file system in an existing virtual block
// device. This function relies on mkfs.X to create the
// file system, the test is skipping if the command is
// not available.
// @param[in] test handle to control the test, i.e., stop it in case of fatal error
// @param[in] type of the file system to be created, e.g., ext3, fat.
// @param[in] path to the virtual block device.
func createFS(t *testing.T, fsType string, path string) {
	cmdBin, lookErr := exec.LookPath("mke2fs")
	if lookErr != nil {
		t.Skip("mke2fs not available, skipping the test")
	}

	var out, err bytes.Buffer
	cmd := exec.Command(cmdBin, "-F", "-t", fsType, path)
	cmd.Stderr = &err
	cmd.Stdout = &out
	cmdErr := cmd.Run()
	if cmdErr != nil {
		t.Fatalf("command failed: %s - stderr: %s - stdout: %s\n", cmdErr, err.String(), out.String())
	}
}

// createFullVirtualBlockDevice creates a full virtual
// block device with a file system in it.
// Warning the file will be earased!!!!
// @param[in] test handle to control the test execution
// @param[in] path to the virtual block device to be created.
// @param[in] type of the file system to be created in the virtual block device (e.g., "ext3")
func createFullVirtualBlockDevice(t *testing.T, path string, fsType string) {
	createVirtualBlockDevice(t, path)
	createFS(t, fsType, path)
}

// ext3InitializerTest prepares the initializer test by
// creating a new virtual block device that will be
// associated with the image, opening the image, get stat
// information about the file associated to the image and
// finally calling the initializer.
// @param[in] test handle to control the test execution.
// @param[in] image handle, when calling, simply an initialized Image variable.
// @param[in] path to the file that will be used with the image; it does not need to exist when calling the function.
// @return the error handle returned by the initializer. We do *not* intend to analyze the handle in this function since only the caller knows if it is a valid or invalid test case
func ext3InitializerTest(t *testing.T, img *Image, path string, fsType string) error {
	createFullVirtualBlockDevice(t, path, fsType)

	var err error
	img.File, err = os.Open(path)
	if err != nil {
		t.Fatalf("cannot open file: %s\n", err)
	}

	fileinfo, err := img.File.Stat()
	if err != nil {
		t.Fatalf("cannot stat image: %s\n", err)
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
	dir, err := ioutil.TempDir("", "headerTesting-")
	if err != nil {
		t.Fatalf("impossible to create temporary directory: %s\n", err)
	}
	defer os.RemoveAll(dir)

	path := dir + "ext3.fs"

	createFullVirtualBlockDevice(t, path, "ext3")

	// Now load the image
	img, imgErr := os.Open(path)
	if imgErr != nil {
		t.Fatal("impossible to load image for testing")
	}
	defer img.Close()
	defer os.Remove(path)

	n, err := img.Read(b)
	if err != nil || n != bufferSize {
		t.Fatalf("cannot read the first %d bytes of the image\n", bufferSize)
	}

	_, checkErr := CheckExt3Header(b)
	if checkErr != nil {
		t.Fatalf("cannot check ext3 header of a valid image: %s\n", checkErr)
	}
}

func TestInitializer(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	// Create a temporary image which is obviously an invalid ext3 image
	f, err := ioutil.TempFile("", "image-")
	if err != nil {
		t.Fatalf("cannot create temporary file: %s\n", err)
	}
	path := f.Name()
	defer os.Remove(path)
	// We do not use defer f.Close() since we will be manually
	// opening and closing the file for testing.
	f.Close()
	resolvedPath, err := ResolvePath(path)
	if err != nil {
		t.Fatalf("failed to retrieve path for %s: %s\n", path, err)
	}

	img := &Image{
		Path: resolvedPath,
		Name: "test",
	}

	img.Writable = true
	var ext3format ext3Format
	mode := ext3format.openMode(true)
	if mode != os.O_RDWR {
		t.Fatal("wrong mode returned")
	}
	img.File, err = os.OpenFile(resolvedPath, mode, 0)
	if err != nil {
		t.Fatalf("cannot open the image: %s\n", err)
	}
	defer img.File.Close()
	fileinfo, err := img.File.Stat()
	if err != nil {
		t.Fatalf("cannot stat image: %s\n", err)
	}

	// This test will fail because we did not set a valid ext3 FS yet
	err = ext3format.initializer(img, fileinfo)
	if err == nil {
		t.Fatal("initializer succeeded while expected to fail")
	}

	// Now we setup a valid ext3 FS and run a test again
	err = ext3InitializerTest(t, img, resolvedPath, "ext3")
	if err != nil {
		t.Fatalf("ext3 initializer test failed with a valid ext3 image: %s\n", err)
	}

	// We now run a test with ext2 to hit some other corner cases
	err = ext3InitializerTest(t, img, resolvedPath, "ext2")
	if err == nil {
		t.Fatal("ext3 initializer test succeeded with an ext2 image while expected to fail")
	}

	// We reformat the image with different file systems and see if we catch
	// the error
	_, lookErr := exec.LookPath("mkfs.fat")
	if lookErr == nil {
		err = ext3InitializerTest(t, img, resolvedPath, "vfat")
		if err == nil {
			t.Fatal("ext3 initializer test succeeded with a vfat image")
		}
	} else {
		t.Log("mkfs.fat command is not available, skipping the test...")
	}

	_, lookErr = exec.LookPath("mkfs.ext4")
	if lookErr == nil {
		err = ext3InitializerTest(t, img, resolvedPath, "ext4")
		if err == nil {
			t.Fatal("ext3 initializer test succeeded with a ext4 image while expected to fail")
		}
	} else {
		t.Log("mkfs.ext4 command is not available, skipping the test...")
	}

	// A small test to exercise openMode() when using read-only mode
	mode = ext3format.openMode(false)
	if mode != os.O_RDONLY {
		t.Fatal("wrong mode returned")
	}

	// Error case when a directory is passed in to initializer()
	path, err = ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("Cannot create a temporary directory: %s\n", err)
	}
	defer os.RemoveAll(path)
	resolvedPath, err = ResolvePath(path)
	if err != nil {
		t.Fatalf("failed to retrieve path for %s: %s\n", resolvedPath, err)
	}
	img.Path = resolvedPath
	img.File, err = os.Open(resolvedPath)
	if err != nil {
		t.Fatalf("cannot open %s: %s\n", resolvedPath, err)
	}
	fileinfo, err = img.File.Stat()
	if err != nil {
		t.Fatalf("cannot stat image: %s\n", err)
	}
	if fileinfo.IsDir() == false {
		t.Fatalf("invalid fileinfo for %s\n", resolvedPath)
	}

	err = ext3format.initializer(img, fileinfo)
	if err == nil {
		t.Fatal("ext3 initializer succeeded with a directory while expected to fail")
	}
}
