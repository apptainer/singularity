// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package types

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewBundle(t *testing.T) {
	invalidDir := "notExsitingDir/"
	validDir := "bundleTests/"
	prefixes := []string{"", "dummyPrefix"}
	testFalseSections := [][]string{{"dummy1", "dummy2", "none"},
		{"none"},
		{"dummy1", "dummy2"}}
	testTrueSections := [][]string{{"all"},
		{"dummy", "all"},
		{"test"},
		{"dummy", "test"}}

	// We create the test directory
	err := os.MkdirAll(validDir, 0755)
	if err != nil {
		t.Fatal("cannot create temporary directory:", validDir)
	}

	// Now we run various tests, it will create directories
	for _, prefix := range prefixes {

		// Tests that should fail
		bundle, myerr := NewBundle(invalidDir, prefix)
		if bundle != nil && myerr == nil {
			t.Fatal("NewBundle() with an invalid directory succeeded while expected to fail")
		}

		// Test that should succeed
		bundle, myerr = NewBundle(validDir, prefix)
		if bundle == nil || myerr != nil {
			t.Fatal("NewBundle() with a valid directory failed while expected to succeed")
		}
		// We check if the directory was actually created
		_, myerr = os.Stat(bundle.Path)
		if myerr != nil {
			t.Fatal("target directory", bundle.Path, "could not be created")
		}
		// With the new bundle, we test Rootfs()
		if bundle.Rootfs() != filepath.Join(bundle.Path, bundle.FSObjects["rootfs"]) {
			t.Fatal("Rootfs() returned the wrong value")
		}
		// And then, we test RunSection()
		for _, falseSection := range testFalseSections {
			bundle.Opts.Sections = falseSection
			if (*bundle).RunSection("notExistingSection") == true {
				t.Fatal("RunSection() returned true while expected to return false")
			}
		}
		for _, trueSection := range testTrueSections {
			bundle.Opts.Sections = trueSection
			if (*bundle).RunSection("test") == false {
				t.Fatal("RunSection() returned false while expected to return true")
			}
		}
		// All done, deleting the directory that was created
		os.RemoveAll(bundle.Path)
	}

	// We delete the test directory
	os.RemoveAll(validDir)
}
