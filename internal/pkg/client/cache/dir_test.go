// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cache

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
	"github.com/sylabs/singularity/pkg/syfs"
)

const cacheCustom = "/tmp/customcachedir"

var expectedCacheCustomRoot = filepath.Join(cacheCustom, CacheDir)
var cacheDefault = filepath.Join(syfs.ConfigDir(), CacheDir)

func TestNewHandle(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	cacheNonExistentParent, err := ioutil.TempDir("", "NonExistingParent-")
	if err != nil {
		t.Fatalf("failed to create a temporary image cache")
	}

	cacheReadOnlyParent, err := fs.MakeTmpDir("", "ReadOnlyParent-", 0555)
	if err != nil {
		t.Fatalf("failed to create a temporary image cache")
	}

	tests := []struct {
		name     string
		dir      string
		expected string
		dropPriv bool
		cleanup  func()
	}{
		{
			name:     "Default root",
			dir:      "",
			expected: cacheDefault,
			// we do not want to change our UID so we can access the correct users home directory
			dropPriv: false,
			// IMPORTANT(ikaneshiro): do nothing here since this is mapped to the users cache
			cleanup: func() {},
		},
		{
			name:     "Custom root",
			dir:      cacheCustom,
			expected: expectedCacheCustomRoot,
			dropPriv: true,
			cleanup: func() {
				os.RemoveAll(cacheCustom)
			},
		},
		{
			name:     "Non-existing root",
			dir:      filepath.Join(cacheNonExistentParent, "NonExistingDir"),
			expected: filepath.Join(cacheNonExistentParent, "NonExistingDir", CacheDir),
			dropPriv: true,
			cleanup: func() {
				os.RemoveAll(cacheNonExistentParent)
			},
		},
		{
			name:     "Read only root",
			dir:      filepath.Join(cacheReadOnlyParent, "NonExistingDir"),
			dropPriv: true,
			cleanup: func() {
				os.RemoveAll(cacheReadOnlyParent)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.dropPriv {
				test.DropPrivilege(t)
				defer test.ResetPrivilege(t)
			}

			c, err := NewHandle(Config{BaseDir: tt.dir})
			if err != nil {
				t.Fatalf("failed to create new image cache handle: %s", err)
			}

			c.checkIfCacheDisabled(t)

			if r := c.rootDir; r != tt.expected {
				t.Errorf("Unexpected result: %s (expected %s)", r, tt.expected)
			}
		})
	}
}

func TestCleanAllCaches(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	imageCacheDir, err := ioutil.TempDir("", "image-cache-")
	if err != nil {
		t.Fatalf("failed to create a temporary image cache")
	}
	defer os.RemoveAll(imageCacheDir)

	c, err := NewHandle(Config{BaseDir: imageCacheDir})
	if err != nil {
		t.Fatalf("failed to create new image cache handle: %s", err)
	}
	c.checkIfCacheDisabled(t)

	// list of subdirs to iterate over
	cacheDirs := map[string]string{
		"library": c.Library,
		"oci":     c.OciTemp,
		"blob":    c.OciBlob,
		"shub":    c.Shub,
		"oras":    c.Oras,
		"net":     c.Net,
	}

	testfile := "test"
	for _, dir := range cacheDirs {
		if err := fs.Touch(filepath.Join(dir, testfile)); err != nil {
			t.Fatalf("Failed to create file in test cache: %v", err)
		}
	}

	// clean out our cache
	c.cleanAllCaches()

	for name, dir := range cacheDirs {
		_, err := os.Stat(filepath.Join(dir, testfile))
		if !os.IsNotExist(err) {
			t.Errorf("Failed to clean %q cache at: %s", name, dir)
		}
	}
}

func TestRoot(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	scratchDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("failed to create temporary directory: %s", err)
	}
	defer os.RemoveAll(scratchDir)

	// We test first with a directory that exists
	existingTempDir, err := ioutil.TempDir(scratchDir, "")
	if err != nil {
		t.Fatalf("failed to create temporary directory: %s", err)
	}
	defer os.RemoveAll(existingTempDir)

	notExistingTempDir := filepath.Join(scratchDir, "dummyDir")

	tests := []struct {
		name           string
		basedir        string
		expectedResult string
	}{
		{
			name:           "existing basedir",
			basedir:        existingTempDir,
			expectedResult: filepath.Join(existingTempDir, CacheDir),
		},
		{
			name:           "nonexisting basedir",
			basedir:        notExistingTempDir,
			expectedResult: filepath.Join(notExistingTempDir, CacheDir),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// drop priv within go routine running this test
			// in order to clean up cache directories without priv
			test.DropPrivilege(t)
			defer test.ResetPrivilege(t)
			imgCache, err := NewHandle(Config{BaseDir: tt.basedir})
			if err != nil {
				t.Fatalf("failed to create new image cache: %s", err)
			}
			imgCache.checkIfCacheDisabled(t)

			root := imgCache.rootDir
			if root != tt.expectedResult {
				t.Fatalf("test %s returned %s instead of %s", tt.name, root, tt.expectedResult)
			}

			cacheBasedir := imgCache.GetBasedir()
			if cacheBasedir != tt.basedir {
				t.Fatalf("image cache basedir for %s is %s instead of %s", tt.name, cacheBasedir, tt.basedir)
			}
		})
	}
}
