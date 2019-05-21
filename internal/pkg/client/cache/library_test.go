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
)

func TestLibrary(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	tests := []struct {
		name     string
		env      string
		expected string
	}{
		{
			name:     "Default Library",
			env:      "",
			expected: filepath.Join(cacheDefault, "library"),
		},
		{
			name:     "Custom Library",
			env:      cacheCustom,
			expected: filepath.Join(cacheCustom, CacheDir, "library"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(DirEnv, tt.env)
			defer os.Unsetenv(DirEnv)

			// This test is based on the default cache so do *not* clean it
			c, err := NewHandle()
			if c == nil || err != nil {
				t.Fatal("failed to create new cache handle")
			}

			if r := c.Library; r != tt.expected {
				t.Errorf("Unexpected result: %s (expected %s)", r, tt.expected)
			}
		})
	}
}

func TestLibraryImage(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	// Create a temporary cache for testing
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("failed to create temporary cache: %s", err)
	}
	defer os.RemoveAll(dir)

	c, err := hdlInit(dir)
	if c == nil || err != nil {
		t.Fatalf("failed to create cache handle: %s", err)
	}
	defer c.Clean("all")

	// Create a dummy entry in the cache to simulate a valid image
	validSHASum := createFakeCachedImage(t, c.Library)

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
			path:     validName,
			expected: filepath.Join(c.Library, validSHASum, validName),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := c.LibraryImage(tt.sum, tt.path)
			if err != nil || path != tt.expected {
				t.Errorf("unexpected result: %s (expected %s)", path, tt.expected)
			}
		})
	}
}

func TestLibraryImageExists(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	// Create a temporary cache for testing
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("failed to create temporary cache: %s", err)
	}
	os.RemoveAll(dir)

	c, err := hdlInit(dir)
	if c == nil || err != nil {
		t.Fatalf("failed to create cache handle: %s", err)
	}
	defer c.Clean("all")

	// Invalid cases
	_, err = c.LibraryImageExists("", "")
	if err == nil {
		t.Fatalf("LibraryImageExists() returned true for invalid data:  %s\n", err)
	}

	// Pull an image so we know for sure that it is in the cache
	if testing.Short() {
		t.Skip("skipping test requiring Singularity to be installed")
	}

	// We build an image in the temporary directory that we manage, which
	// will also cache it in the cache sub-directory in the same
	// temporary directory
	sexec, err := exec.LookPath("singularity")
	if err != nil {
		t.Skip("skipping test: singularity is not installed")
	}
	filename := "alpine_latest.sif"
	name := filepath.Join(dir, filename)
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(sexec, "build", "-F", name, "library://alpine")
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	err = cmd.Run()
	if err != nil {
		t.Fatalf("command failed: %s - stdout: %s - stderr: %s\n", err, stdout.String(), stderr.String())
	}

	// Invalid case with a valid image
	_, err = c.LibraryImageExists("", filename)
	if err == nil {
		t.Fatalf("invalid case with a valid image succeeded: %s\n", err)
	}

	// Valid case with a valid image, the get the hash from the
	// cache we just created and check whether it matches with what
	// we have in the cache
	subdirs, err := ioutil.ReadDir(c.Library)
	if err != nil || len(subdirs) != 1 {
		t.Fatalf("failed to find the cache directory (%d directories found in %s)", len(subdirs), c.Library)
	}
	hash := subdirs[0].Name()

	exists, err := c.LibraryImageExists(hash, filename)
	if err != nil {
		t.Fatalf("error while checking if image exists: %s\n", err)
	}
	if exists == false {
		t.Fatalf("valid image (%s) is reported as non-existing", filename)
	}
}
