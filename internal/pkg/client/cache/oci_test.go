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

	if os.Getenv(DisableEnv) == "1" {
		t.Skip("Caching is disabled")
	}

	tests := []struct {
		name        string
		dir         string
		needCleanup bool
		expected    string
	}{
		{
			name:        "Default OCI blob",
			dir:         "",
			needCleanup: false, // Never clean up the default cache
			expected:    filepath.Join(cacheDefault, "oci"),
		},
		{
			name:        "Custom OCI blob",
			dir:         cacheCustom,
			needCleanup: true,
			expected:    filepath.Join(expectedCacheCustomRoot, "oci"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewHandle(tt.dir)
			if err != nil {
				t.Fatalf("failed to create new image cache handle: %s", err)
			}
			if tt.needCleanup {
				defer c.cleanAllCaches()
			}

			if c.OciBlob != tt.expected {
				t.Errorf("Unexpected result: %s (expected %s)", c.OciBlob, tt.expected)
			}
		})
	}
}

func TestOciTemp(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	if os.Getenv(DisableEnv) == "1" {
		t.Skip("Caching is disabled")
	}

	tests := []struct {
		name        string
		dir         string
		needCleanup bool
		expected    string
	}{
		{
			name:        "Default OCI temp",
			dir:         "",
			needCleanup: false, // Never clean up default cache
			expected:    filepath.Join(cacheDefault, "oci-tmp"),
		},
		{
			name:        "Custom OCI temp",
			dir:         cacheCustom,
			needCleanup: true,
			expected:    filepath.Join(expectedCacheCustomRoot, "oci-tmp"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewHandle(tt.dir)
			if err != nil {
				t.Fatalf("failed to create new image cache handle: %s", err)
			}
			if tt.needCleanup {
				defer c.cleanAllCaches()
			}

			if c.OciTemp != tt.expected {
				t.Errorf("Unexpected result: %s (expected %s)", c.OciTemp, tt.expected)
			}
		})
	}
}

func TestOciTempExists(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	if os.Getenv(DisableEnv) == "1" {
		t.Skip("Caching is disabled")
	}

	tempImageCache, err := ioutil.TempDir("", "image-cache-")
	if err != nil {
		t.Fatal("failed to create temporary image cache directory:", err)
	}
	defer os.RemoveAll(tempImageCache)

	c, err := NewHandle(tempImageCache)
	if err != nil {
		t.Fatalf("failed to create new image cache handle: %s", err)
	}

	tests := []struct {
		name     string
		sum      string
		path     string
		expected bool
	}{
		{
			name:     "empty",
			sum:      validSHASum,
			path:     validPath,
			expected: true,
		},
		{
			name:     "invalid SHA sum; valid path",
			sum:      invalidSHASum,
			path:     validPath,
			expected: true,
		},
		{
			name:     "valid SHA sum; invalid path",
			sum:      validSHASum,
			path:     invalidPath,
			expected: false,
		},
		{
			name:     "invalid",
			sum:      invalidSHASum,
			path:     invalidPath,
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			exists, err := c.OciTempExists(test.sum, test.path)
			if err != nil {
				t.Fatalf("OciTempExists() failed: %s\n", err)
			}
			if test.expected == true && exists == false {
				t.Fatal("test expected to succeed but failed")
			}
			if test.expected == false && exists == true {
				t.Fatal("test expected to fail but succeeded")
			}
		})
	}
}
