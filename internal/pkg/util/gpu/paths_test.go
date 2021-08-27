// Copyright (c) 2020-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

//go:build linux
// +build linux

package gpu

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const testLibFile = "testdata/testliblist.conf"

var testLibList = []string{"libc.so", "echo"}

func TestElfMachine(t *testing.T) {
	gotMachine, err := elfMachine()
	if err != nil {
		t.Errorf("elfMachine() error = %v", err)
		return
	}
	if gotMachine <= 0 {
		t.Errorf("elfMachine() gotMachine = %v is <=0", gotMachine)
	}
}

func TestLdCache(t *testing.T) {
	gotCache, err := ldCache()
	if err != nil {
		t.Errorf("ldCache() error = %v", err)
		return
	}
	if len(gotCache) == 0 {
		t.Error("ldCache() gave no results")
	}
	for path, name := range gotCache {
		if strings.HasPrefix(name, "ld-linux") {
			if strings.Contains(path, "ld-linux") {
				return
			}
		}
	}
	t.Error("ldCache() result did not include expected ld-linux entry")
}

func Test_gpuliblist(t *testing.T) {
	gotLibs, err := gpuliblist(testLibFile)
	if err != nil {
		t.Errorf("gpuliblist() error = %v", err)
		return
	}
	if len(gotLibs) == 0 {
		t.Error("gpuliblist() gave no results")
	}
	if !reflect.DeepEqual(gotLibs, testLibList) {
		t.Errorf("gpuliblist() gave unexpected results, got: %v expected: %v", gotLibs, testLibList)
	}
}

func TestSoLinks(t *testing.T) {
	// Test link structure:
	// a.so.1.2 -> a.so.1 -> a.so (file)
	//   - soLinks(a.so) should give both of these symlinks
	// a.so.2 -> b.so
	//   - this should *not* get included, as it doesn't resolve back to a.so
	tmpDir, err := ioutil.TempDir("", "test-solinks")
	if err != nil {
		t.Fatalf("Could not create tempDir: %v", err)
	}
	aFile := filepath.Join(tmpDir, "a.so")
	a1Link := filepath.Join(tmpDir, "a.so.1")
	a12Link := filepath.Join(tmpDir, "a.so.1.2")
	if err := ioutil.WriteFile(aFile, nil, 0o644); err != nil {
		t.Fatalf("Could not create file: %v", err)
	}
	if err := os.Symlink(aFile, a1Link); err != nil {
		t.Fatalf("Could not symlink: %v", err)
	}
	if err := os.Symlink(aFile, a12Link); err != nil {
		t.Fatalf("Could not symlink: %v", err)
	}
	bFile := filepath.Join(tmpDir, "b.so")
	err = ioutil.WriteFile(bFile, nil, 0o644)
	if err != nil {
		t.Fatalf("Could not create file: %v", err)
	}
	a2Link := filepath.Join(tmpDir, "a.so.2")
	if err := os.Symlink(bFile, a2Link); err != nil {
		t.Fatalf("Could not symlink: %v", err)
	}

	expectedLinks := []string{a1Link, a12Link}

	gotLinks, err := soLinks(aFile)
	if err != nil {
		t.Errorf("soLinks() error = %v", err)
		return
	}
	if len(gotLinks) == 0 {
		t.Error("soLinks() gave no results")
	}
	if !reflect.DeepEqual(gotLinks, expectedLinks) {
		t.Errorf("soList() gave unexpected results, got: %v expected: %v", gotLinks, expectedLinks)
	}
}

func TestPaths(t *testing.T) {
	// Very naive sanity test. Check we can find one lib and one binary without error
	gotLibs, gotBin, err := paths(testLibList)
	if err != nil {
		t.Errorf("paths() error = %v", err)
		return
	}
	if len(gotLibs) == 0 {
		t.Error("paths() gave no libraries")
	}
	if len(gotBin) == 0 {
		t.Error("paths() gave no binaries")
	}
}
