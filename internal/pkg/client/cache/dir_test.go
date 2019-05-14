// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cache

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
)

func TestRoot(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	expectedDefaultRoot, expectedCustomRoot := getDefaultCacheValues(t)

	tests := []struct {
		name     string
		env      string
		expected string
	}{
		{
			name:     "Default root",
			env:      "",
			expected: expectedDefaultRoot,
		},
		{
			name:     "Custom root",
			env:      cacheCustom,
			expected: expectedCustomRoot,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(DirEnv, tt.env)
			defer os.Unsetenv(DirEnv)

			newCache := setupCache(t)
			if newCache == nil {
				t.Fatal("failed to create temporary cache")
			}
			defer cleanupCache(t, newCache)

			if r := newCache.rootDir; r != tt.expected {
				t.Errorf("Unexpected result: %s (expected %s)", r, tt.expected)
			}
		})
	}
}

func testFakeCacheSubdir(t *testing.T, basedir string, subdir string) {
	// Invalid cache: we create a file instead of a directory
	cachePath := filepath.Join(basedir, rootDefault, subdir)

	// If the path exists, delete it so we can start from a clean state
	exists, _ := fs.Exists(cachePath)
	if exists == true {
		err := os.Remove(cachePath)
		if err != nil {
			t.Fatalf("failed to delete %s: %s", cachePath, err)
		}
	}

	// Create the file
	err := fs.Touch(cachePath)
	if err != nil {
		t.Fatalf("failed to create %s: %s", cachePath, err)
	}

	// Run the test
	newCache, err := hdlInit(basedir)
	if err == nil || newCache != nil {
		if err == nil {
			fmt.Println("No error")
		}
		if newCache != nil {
			fmt.Println("cache created")
		}
		t.Fatal("cache creation from invalid data succeeded")
	}

	// Clean up by removing the file we created
	err = os.Remove(cachePath)
	if err != nil {
		t.Fatalf("failed to delete %s: %s", cachePath, err)
	}
}

// TestCreate creates a temporary file that is then used as base directory
// for a new cache. This lets us have a fine-grain control over the test
// (including switching some of the cache's directories to read-only), without
// polluting the actual cache that the person running the test may have
// specified via the DirEnv environment variable for another context.
// This also allows us to know for sure that the cache is NOT already there
// and therefore executed in a clean setup.
// In other words, we try to control the test settings as much as possible to
// run low-level tests related to cache creation.
func TestCreate(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal("failed to create temporary directory")
	}

	// Create the root of our fake cache
	rootCachePath := filepath.Join(dir, rootDefault)
	err = os.MkdirAll(rootCachePath, 0777)
	if err != nil {
		t.Fatalf("Failed to create directory %s: %s", rootCachePath, err)
	}

	tests := []struct {
		name   string
		subdir string
	}{
		{
			name:   "Invalid Shub cache",
			subdir: ShubDir,
		},
		{
			name:   "Invalid Net cache",
			subdir: NetDir,
		},
		{
			name:   "Invalid OCi blob cache",
			subdir: OciBlobDir,
		},
		{
			name:   "Invalid OCI temp cache",
			subdir: OciTempDir,
		},
		{
			name:   "Invalid library cache",
			subdir: LibraryDir,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// testFakeCacheSubdir will fail the test if an error occurs and
			// as a result, we do not need to check for errors here.
			testFakeCacheSubdir(t, dir, tt.subdir)
		})
	}

	// Another error case: the cache's root is not writable for the user
	// Change the mode of root
	err = os.Chmod(dir, 0444)
	if err != nil {
		t.Fatalf("cannot change mode of root: %s", err)
	}

	tempCache, err := hdlInit(dir)
	if err == nil || tempCache != nil {
		t.Fatal("cache creation from invalid data succeeded")
	}
}

// TestIsValid is only testing corner cases, i.e., invalid cases
// of the IsValid(). Valid cases are covered by other tests.
func TestIsValid(t *testing.T) {
	test.DropPrivilege(t)
	test.ResetPrivilege(t)

	tempCache := createTempCache(t)
	if tempCache == nil {
		t.Fatal("cannot create cache")
	}
	defer tempCache.Clean()

	validBasedir := tempCache.BaseDir
	validRoot := tempCache.rootDir
	validLibrary := tempCache.Library
	validNet := tempCache.Net
	validOcitemp := tempCache.OciTemp
	validOciblob := tempCache.OciBlob
	validShub := tempCache.Shub

	tests := []struct {
		name    string
		basedir string
		root    string
		library string
		net     string
		ocitemp string
		ociblob string
		shub    string
	}{
		{
			name:    "empty basedir",
			basedir: "",
			root:    validRoot,
			library: validLibrary,
			net:     validNet,
			ocitemp: validOcitemp,
			ociblob: validOciblob,
			shub:    validShub,
		},
		{
			name:    "empty root",
			basedir: validBasedir,
			root:    "",
			library: validLibrary,
			net:     validNet,
			ocitemp: validOcitemp,
			ociblob: validOciblob,
			shub:    validShub,
		},
		{
			name:    "empty library",
			basedir: validBasedir,
			root:    validRoot,
			library: "",
			net:     validNet,
			ocitemp: validOcitemp,
			ociblob: validOciblob,
			shub:    validShub,
		},
		{
			name:    "empty net",
			basedir: validBasedir,
			root:    validRoot,
			library: validLibrary,
			net:     "",
			ocitemp: validOcitemp,
			ociblob: validOciblob,
			shub:    validShub,
		},
		{
			name:    "empty ocitemp",
			basedir: validBasedir,
			root:    validRoot,
			library: validLibrary,
			net:     validNet,
			ocitemp: "",
			ociblob: validOciblob,
			shub:    validShub,
		},
		{
			name:    "empty ociblob",
			basedir: validBasedir,
			root:    validRoot,
			library: validLibrary,
			net:     validNet,
			ocitemp: validOcitemp,
			ociblob: "",
			shub:    validShub,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempCache.BaseDir = tt.basedir
			tempCache.rootDir = tt.root
			tempCache.Library = tt.library
			tempCache.Net = tt.net
			tempCache.OciTemp = tt.ocitemp
			tempCache.OciBlob = tt.ociblob
			tempCache.Shub = tt.shub
			isValid := tempCache.IsValid()
			if isValid {
				t.Fatal("invalid cache data was designated as valid")
			}
		})
	}
}

// TestUpdateCacheSubdir is only testing corner cases, i.e., invalid cases
// of the updateCacheSubdir(). Valid cases are covered by other tests.
func TestUpdateCacheSubdir(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	tempCache := createTempCache(t)
	if tempCache == nil {
		t.Fatal("failed to create temporary cache")
	}
	defer tempCache.Clean()

	tests := []struct {
		name string
		c    *SingularityCache
		dir  string
	}{
		{
			name: "undefined cache",
			c:    nil,
			dir:  "aCacheType",
		},
		{
			name: "empty dir",
			c:    tempCache,
			dir:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := updateCacheSubdir(tt.c, tt.dir)
			if err == nil {
				t.Fatal("successfully updated the cache's sub-directory with invalid parameter")
			}
		})
	}
}
