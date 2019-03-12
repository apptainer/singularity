// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package unpacker

import (
	"bufio"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func createArchive(t *testing.T) *os.File {
	mk, err := exec.LookPath("mksquashfs")
	if err != nil {
		t.SkipNow()
	}
	f, err := ioutil.TempFile("", "archive-")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(mk, ".", f.Name(), "-noappend", "-no-progress")
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	return f
}

func isExist(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func TestSquashfs(t *testing.T) {
	s := NewSquashfs()

	if !s.HasUnsquashfs() {
		t.SkipNow()
	}

	dir, err := ioutil.TempDir("", "unpacker-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// create archive with files present in this directory
	archive := createArchive(t)
	defer os.Remove(archive.Name())

	savedPath := s.UnsquashfsPath

	// test with an empty unsquashfs path
	s.UnsquashfsPath = ""
	if err := s.ExtractAll(archive, dir); err == nil {
		t.Errorf("unexpected success with empty unsquashfs path")
	}
	// test with a bad unsquashfs path
	s.UnsquashfsPath = "/unsquashfs-no-exists"
	if err := s.ExtractAll(archive, dir); err == nil {
		t.Errorf("unexpected success with bad unsquashfs path")
	}

	s.UnsquashfsPath = savedPath

	// extract all into temporary folder
	if err := s.ExtractAll(archive, dir); err != nil {
		t.Error(err)
	}

	// check if squashfs.go was extracted
	path := filepath.Join(dir, "squashfs.go")
	if !isExist(path) {
		t.Errorf("extraction failed, %s is missing", path)
	}
	os.Remove(path)

	// check if squashfs_test.go was extracted
	path = filepath.Join(dir, "squashfs_test.go")
	if !isExist(path) {
		t.Errorf("extraction failed, %s is missing", path)
	}
	os.Remove(path)

	// test with an empty file list
	if err := s.ExtractFiles([]string{}, archive, dir); err == nil {
		t.Errorf("unexpected success with empty file list")
	}

	// extract squashfs_test.go only
	if err := s.ExtractFiles([]string{"squashfs_test.go"}, bufio.NewReader(archive), dir); err != nil {
		t.Error(err)
	}
	// check that squashfs.go was not extracted
	path = filepath.Join(dir, "squashfs.go")
	if isExist(path) {
		t.Errorf("file extraction failed, %s is present", path)
	}
	// check that squashfs_test.go was extracted
	path = filepath.Join(dir, "squashfs_test.go")
	if !isExist(path) {
		t.Errorf("file extraction failed, %s is missing", path)
	}
}
