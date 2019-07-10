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
	"github.com/sylabs/singularity/pkg/syfs"
)

const cacheCustom = "/tmp/customcachedir"

var expectedCacheCustomRoot = filepath.Join(cacheCustom, CacheDir)
var cacheDefault = filepath.Join(syfs.ConfigDir(), CacheDir)

func TestCleanAllCaches(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	tests := []struct {
		name     string
		dir      string
		expected string
	}{
		{
			name:     "Default root",
			dir:      "",
			expected: cacheDefault,
		},
		{
			name:     "Custom root",
			dir:      cacheCustom,
			expected: expectedCacheCustomRoot,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewHandle(tt.dir)
			if err != nil {
				t.Fatalf("failed to create new image cache handle: %s", err)
			}
			/* This is evil: if the cache is the default cache, we clean it */
			defer c.cleanAllCaches()

			if r := c.rootDir; r != tt.expected {
				t.Errorf("Unexpected result: %s (expected %s)", r, tt.expected)
			}
		})
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
			imgCache, err := NewHandle(tt.basedir)
			if err != nil {
				t.Fatalf("failed to create new image cache: %s", err)
			}

			root := imgCache.Root()
			if root == tt.expectedResult {
				t.Fatalf("test %s returned %s instead of %s", tt.name, root, tt.expectedResult)
			}

			cacheBasedir := imgCache.GetBasedir()
			if cacheBasedir != tt.basedir {
				t.Fatalf("image cache basedir for %s is %s instead of %s", tt.name, cacheBasedir, tt.basedir)
			}
		})
	}
}
