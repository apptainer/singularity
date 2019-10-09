// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package packer

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func checkArchive(t *testing.T, path string, files []string) {
	un, err := exec.LookPath("unsquashfs")
	if err != nil {
		t.SkipNow()
	}

	dir, err := ioutil.TempDir("", "extracted-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	cmd := exec.Command(un, "-f", "-d", dir, path)
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	for _, f := range files {
		path := filepath.Join(dir, f)
		if !isExist(path) {
			t.Errorf("squashfs verification failed: %s is missing", path)
		}
	}
}

func isExist(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func createSquashfs(t *testing.T, s *Squashfs) (string, error) {
	image, err := ioutil.TempFile("", "packer-")
	if err != nil {
		t.Fatal(err)
	}
	image.Close()

	err = s.Create([]string{"."}, image.Name(), []string{"-noappend"})

	return image.Name(), err
}

func testEmptyMksquashfsPath(t *testing.T) {
	s := NewSquashfs()
	s.MksquashfsPath = ""

	imageName, err := createSquashfs(t, s)
	defer os.Remove(imageName)

	if err == nil {
		t.Errorf("unexpected success with empty mksquashfs path")
	}
}

func testInvalidMksquashfsPath(t *testing.T) {
	s := NewSquashfs()
	s.MksquashfsPath = "/mksquashfs-no-exists"

	imageName, err := createSquashfs(t, s)
	defer os.Remove(imageName)

	if err == nil {
		t.Errorf("unexpected success with bad mksquashfs path")
	}
}

func testNonZeroExitCode(t *testing.T) {
	s := NewSquashfs()
	s.MksquashfsPath, _ = exec.LookPath("false")

	imageName, err := createSquashfs(t, s)
	defer os.Remove(imageName)

	if err == nil {
		t.Errorf("unexpected success with non-zero exit code")
	}
}

func testHappyPath(t *testing.T) {
	s := NewSquashfs()

	imageName, err := createSquashfs(t, s)
	defer os.Remove(imageName)

	if err != nil {
		t.Error(err)
	}

	// XXX(mem): this test will fail if we modify the contents of
	// this directory
	//
	// ensure we can extract these files from squashfs
	checkArchive(t, imageName, []string{"squashfs.go", "squashfs_test.go"})
}

func TestSquashfs(t *testing.T) {
	if s := NewSquashfs(); !s.HasMksquashfs() {
		t.Skip("mksquashfs not found, skipping")
	}

	t.Run("empty mksquashfs path", testEmptyMksquashfsPath)
	t.Run("invalid mksquashfs path", testInvalidMksquashfsPath)
	t.Run("non-zero exit code", testNonZeroExitCode)
	t.Run("happy path", testHappyPath)
}
