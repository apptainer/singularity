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

func TestSquashfs(t *testing.T) {
	s := NewSquashfs()

	if !s.HasMksquashfs() {
		t.SkipNow()
	}

	image, err := ioutil.TempFile("", "packer-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(image.Name())

	savedPath := s.MksquashfsPath
	// test with an empty unsquashfs path
	s.MksquashfsPath = ""
	if err := s.Create([]string{"."}, image.Name(), []string{"-noappend"}); err == nil {
		t.Errorf("unexpected success with empty unsquashfs path")
	}
	// test with a bad mksquashfs path
	s.MksquashfsPath = "/mksquashfs-no-exists"
	if err := s.Create([]string{"."}, image.Name(), []string{"-noappend"}); err == nil {
		t.Errorf("unexpected success with bad unsquashfs path")
	}

	s.MksquashfsPath = savedPath

	// create squashfs in temporary file
	if err := s.Create([]string{"."}, image.Name(), []string{"-noappend"}); err != nil {
		t.Error(err)
	}

	// ensure we can extract these files from squashfs
	checkArchive(t, image.Name(), []string{"squashfs.go", "squashfs_test.go"})
}
