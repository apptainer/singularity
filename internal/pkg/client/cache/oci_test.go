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

func TestOciBlob(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	tests := []struct {
		name     string
		env      string
		expected string
	}{
		{
			name:     "Default OCI blob",
			env:      "",
			expected: filepath.Join(cacheDefault, "oci"),
		},
		{
			name:     "Custom OCI blob",
			env:      cacheCustom,
			expected: filepath.Join(cacheCustom, CacheDir, "oci"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(DirEnv, tt.env)
			defer os.Unsetenv(DirEnv)

			// This test is based on the default cache, do not clean up
			c, err := NewHandle()
			if c == nil || err != nil {
				t.Fatal("failed to create cache handle")
			}

			if c.OciBlob != tt.expected {
				t.Fatalf("unexpected result: %s (expected %s)", c.OciBlob, tt.expected)
			}
		})
	}
}

func TestOciTemp(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	tests := []struct {
		name     string
		env      string
		expected string
	}{
		{
			name:     "Default OCI temp",
			env:      "",
			expected: filepath.Join(cacheDefault, "oci-tmp"),
		},
		{
			name:     "Custom OCI temp",
			env:      cacheCustom,
			expected: filepath.Join(cacheCustom, CacheDir, "oci-tmp"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(DirEnv, tt.env)
			defer os.Unsetenv(DirEnv)

			// This test is based on the default cache, do not clean it
			c, err := NewHandle()
			if c == nil || err != nil {
				t.Fatal("failed to create cache handle")
			}

			if c.OciTemp != tt.expected {
				t.Fatalf("Unexpected result: %s (expected %s)", c.OciTemp, tt.expected)
			}
		})
	}
}

func TestOciTempExists(t *testing.T) {
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

	validSHASum := createFakeCachedImage(t, c.OciTemp)

	tests := []struct {
		name      string
		sum       string
		path      string
		expected  bool
		shallPass bool
	}{
		{
			name:      "empty",
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
			name:      "invalid",
			sum:       invalidSHASum,
			path:      invalidName,
			expected:  false,
			shallPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists, err := c.OciTempExists(tt.sum, tt.path)
			if tt.shallPass == true && (exists != tt.expected || err != nil) {
				t.Fatalf("%s expected to succeed but failed", tt.name)
			}
			if tt.shallPass == false && (exists != tt.expected && err != nil) {
				t.Fatal("tt expected to fail but succeeded")
			}
		})
	}
}
