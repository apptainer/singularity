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

	client "github.com/sylabs/scs-library-client/client"
	"github.com/sylabs/singularity/internal/pkg/test"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
)

func TestLibrary(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	tests := []struct {
		name        string
		dir         string
		needCleanup bool
		expected    string
	}{
		{
			name:        "Default Library",
			dir:         "",
			needCleanup: false, // Never clean up the default cache
			expected:    filepath.Join(cacheDefault, "library"),
		},
		{
			name:        "Custom Library",
			dir:         cacheCustom,
			needCleanup: true,
			expected:    filepath.Join(expectedCacheCustomRoot, "library"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewHandle(tt.dir)
			if c == nil || err != nil {
				t.Fatal("failed to create new image cache handle")
			}
			if tt.needCleanup {
				defer c.cleanAllCaches()
			}

			if c.Library != tt.expected {
				t.Errorf("unexpected result for test %s: %s (expected %s)", tt.name, c.Library, tt.expected)
			}
		})
	}
}

func TestLibraryImage(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	tempImageCache, err := ioutil.TempDir("", "image-cache-")
	if err != nil {
		t.Fatal("failed to create temporary image cache")
	}
	defer os.RemoveAll(tempImageCache)

	c, err := NewHandle(tempImageCache)
	if c == nil || err != nil {
		t.Fatal("failed to create new image cache handle")
	}

	// LibraryImage just return a string and there is no definition of what
	// could be a bad string.
	tests := []struct {
		name     string
		sum      string
		path     string
		expected string
	}{
		{
			name:     "General case",
			sum:      validSHASum,
			path:     validPath,
			expected: c.Library,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := c.LibraryImage(tt.sum, tt.path)
			if path != tt.expected {
				t.Errorf("unexpected result: %s (expected %s)", path, tt.expected)
			}
		})
	}
}

func createValidFakeImageInCache(t *testing.T, c *Handle) (string, string, string) {
	filename := "dummyImage.sif"
	// At first we assume the hash is "0" and will be updated after the file is actually created
	sum := "0"
	sumPath := filepath.Join(c.Library, sum)
	err := os.MkdirAll(sumPath, 0755)
	if err != nil {
		t.Fatalf("failed to create directory %s: %s", sumPath, err)
	}
	destPath := filepath.Join(sumPath, filename)
	err = fs.Touch(destPath) // The file will be automatically deteled when cleaning the cache
	if err != nil {
		t.Fatalf("failed to create empty file %s: %s", destPath, err)
	}

	// Calculate the actual hash of the file
	sum, err = client.ImageHash(destPath)
	if err != nil {
		t.Fatalf("failed to get hash for image %s: %s", destPath, err)
	}

	// Update the path to the file with the valid hash value
	newSumPath := filepath.Join(c.Library, sum)
	err = os.Rename(sumPath, newSumPath)
	if err != nil {
		t.Fatalf("failed to rename directory %s to %s: %s", sumPath, newSumPath, err)
	}

	fmt.Println("File is:", filepath.Join(newSumPath, filename))
	return filename, filepath.Join(newSumPath, filename), sum
}

func TestLibraryImageExists(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	imageCacheDir, err := ioutil.TempDir("", "image-cache-")
	if err != nil {
		t.Fatal("failed to create temporary image cache directory")
	}
	//defer os.RemoveAll(imageCacheDir)
	fmt.Println("Cache is there: ", imageCacheDir)
	c, err := NewHandle(imageCacheDir)
	if c == nil || err != nil {
		t.Fatal("failed to create new image cache handle")
	}

	// Invalid cases
	_, err = c.LibraryImageExists("", "")
	if err == nil {
		t.Fatalf("LibraryImageExists() returned true for invalid data:  %s\n", err)
	}

	// Pull an image so we know for sure that it is in the cache
	if testing.Short() {
		t.Skip("skipping test requiring Singularity to be installed")
	}

	imgName, imgPath, hash := createValidFakeImageInCache(t, c)

	// Invalid case with a valid image
	_, err = c.LibraryImageExists("", imgPath)
	if err != nil {
		t.Fatalf("image %s reported as non-existing: %s\n", imgPath, err)
	}

	// Valid case with a valid image, the get the hash from the
	// file we just created and check whether it matches with what
	// we have in the cache
	exists, err := c.LibraryImageExists(hash, imgName)
	if err != nil {
		t.Fatalf("error while checking if image exists: %s\n", err)
	}
	if exists == false {
		t.Fatalf("valid image %s is reported as non-existing", imgPath)
	}
}
