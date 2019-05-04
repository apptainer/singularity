// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cache

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
	client "github.com/sylabs/singularity/pkg/client/library"
)

func TestLibrary(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	expectedDefaultCache, expectedCustomCache := getDefaultCacheValues(t)

	expectedDefaultLibCache := filepath.Join(expectedDefaultCache, "library")
	expectedCustomLibCache := filepath.Join(expectedCustomCache, "library")

	tests := []struct {
		name     string
		env      string
		expected string
	}{
		{
			name:     "Default Library",
			env:      "",
			expected: expectedDefaultLibCache,
		},
		{
			name:     "Custom Library",
			env:      cacheCustom,
			expected: expectedCustomLibCache,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(DirEnv, tt.env)
			defer os.Unsetenv(DirEnv)

			newCache := createTempCache(t)
			if newCache == nil {
				t.Fatal("failed to create temporary cache")
			}
			defer newCache.Clean()

			if r, err := newCache.Library(); err != nil || r != tt.expected {
				t.Errorf("Unexpected result: %s (expected %s)", r, tt.expected)
			}
		})
	}
}

func TestLibraryImage(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	newCache := CreateTempCache(t)
	if newCache == nil {
		t.Fatal("failed to create temporary cache")
	}
	defer newCache.Clean()

	expectedCache, err := newCache.Library()
	if err != nil {
		t.Fatalf("failed to get library cache's path: %s", err)
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
			expected: expectedCache,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := newCache.LibraryImage(tt.sum, tt.path)
			if err != nil || path != tt.expected {
				t.Errorf("unexpected result: %s (expected %s)", path, tt.expected)
			}
		})
	}
}

func TestLibraryImageExists(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	newCache := CreateTempCache(t)
	defer newCache.Clean()

	// Invalid cases
	_, err := newCache.LibraryImageExists("", "")
	if err == nil {
		t.Fatalf("LibraryImageExists() returned true for invalid data:  %s\n", err)
	}

	// Pull an image so we know for sure that it is in the cache
	if testing.Short() {
		t.Skip("skipping test requiring Singularity to be installed")
	}

	sexec, err := exec.LookPath("singularity")
	if err != nil {
		t.Skip("skipping test: singularity is not installed")
	}
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("cannot create temporary directory: %s\n", err)
	}
	defer os.RemoveAll(dir)
	filename := "alpine_latest.sif"
	name := filepath.Join(dir, filename)
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(sexec, "pull", "-F", "-U", name, "library://alpine")
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	err = cmd.Run()
	if err != nil {
		t.Fatalf("command failed: %s - stdout: %s - stderr: %s\n", err, stdout.String(), stderr.String())
	}

	// Invalid case with a valid image
	_, err = newCache.LibraryImageExists("", filename)
	if err != nil {
		t.Fatalf("image reported as non-existing: %s\n", err)
	}

	// Valid case with a valid image, the get the hash from the
	// file we just created and check whether it matches with what
	// we have in the cache
	hash, err := client.ImageHash(name)
	if err != nil {
		t.Fatalf("cannot get image's hash: %s\n", err)
	}

	exists, err := newCache.LibraryImageExists(hash, filename)
	if err != nil {
		t.Fatalf("error while checking if image exists: %s\n", err)
	}
	if exists == false {
		t.Fatal("valid image is reported as non-existing")
	}
}
