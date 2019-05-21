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
)

func TestNet(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	tests := []struct {
		name     string
		env      string
		expected string
	}{
		{
			name:     "Default Net",
			env:      "",
			expected: filepath.Join(cacheDefault, "net"),
		},
		{
			name:     "Custom Net",
			env:      cacheCustom,
			expected: filepath.Join(cacheCustom, CacheDir, "net"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(DirEnv, tt.env)
			defer os.Unsetenv(DirEnv)

			// This test uses the default cache; do not clean it
			c, err := NewHandle()
			if c == nil || err != nil {
				t.Fatalf("failed to create a cache handle")
			}

			if c.Net != tt.expected {
				t.Errorf("Unexpected result: %s (expected %s)", c.Net, tt.expected)
			}
		})
	}
}

func TestNetImageExists(t *testing.T) {
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
	validSHASum := createFakeCachedImage(t, c.Net)

	tests := []struct {
		name      string
		sum       string
		path      string
		expected  bool
		shallPass bool
	}{
		{
			name:      "valid data",
			sum:       validSHASum,
			path:      validName,
			expected:  true,
			shallPass: true,
		},
		{
			name:      "invalid SHA sum; valid path",
			sum:       invalidSHASum,
			path:      validName,
			expected:  false,
			shallPass: false,
		},
		{
			name:      "valid SHA sum; invalid path",
			sum:       validSHASum,
			path:      invalidName,
			expected:  false,
			shallPass: false,
		},
		{
			name:      "invalid data",
			sum:       invalidSHASum,
			path:      invalidName,
			expected:  false,
			shallPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists, err := c.NetImageExists(tt.sum, tt.path)
			if tt.shallPass == false && (exists != tt.expected || err == nil) {
				t.Fatal("NetImageExists() did not fail for an invalid case")
			}
			if tt.shallPass == true && (exists != tt.expected || err != nil) {
				t.Fatalf("NetImageExists() failed while expected to succeed (expected: %v)", tt.expected)
			}
		})
	}
}
