// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cache

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
)

func TestOciBlob(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	expectedDefaultCache, expectedCustomCache := getDefaultCacheValues(t)

	tests := []struct {
		name     string
		env      string
		expected string
	}{
		{
			name:     "Default OCI blob",
			env:      "",
			expected: filepath.Join(expectedDefaultCache, "oci"),
		},
		{
			name:     "Custom OCI blob",
			env:      cacheCustom,
			expected: filepath.Join(expectedCustomCache, "oci"),
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

			if newCache.OciBlob != tt.expected {
				t.Errorf("Unexpected result: %s (expected %s)", newCache.OciBlob, tt.expected)
			}
		})
	}
}

func TestOciTemp(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	expectedDefaultCache, expectedCustomCache := getDefaultCacheValues(t)

	tests := []struct {
		name     string
		env      string
		expected string
	}{
		{
			name:     "Default OCI temp",
			env:      "",
			expected: filepath.Join(expectedDefaultCache, "oci-tmp"),
		},
		{
			name:     "Custom OCI temp",
			env:      cacheCustom,
			expected: filepath.Join(expectedCustomCache, "oci-tmp"),
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

			if newCache.OciTemp != tt.expected {
				t.Errorf("Unexpected result: %s (expected %s)", newCache.OciTemp, tt.expected)
			}
		})
	}
}

func TestOciTempImage(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	// Now a few error cases that need explicit testing
	invalidCache := createTempCache(t)
	if invalidCache == nil {
		t.Fatal("unable to create temporary cache")
	}
	defer invalidCache.Clean()
	invalidCache.State = StateInvalid
	_, err := invalidCache.OciTempExists(validSHASum, validPath)
	if err == nil {
		t.Fatal("OciTempExists() on an invalid cache succeeded")
	}

	_, err = invalidCache.OciTempImage(validSHASum, validPath)
	if err == nil {
		t.Fatal("OciTempImage() on a invalid cache succeeded")
	}

	// we change the access mode of the cache's root to reach a specific error case
	validCache := createTempCache(t)
	if validCache == nil {
		t.Fatal("unable to create temporary cache")
	}
	defer validCache.Clean()
	err = os.Chmod(validCache.rootDir, 0444)
	if err != nil {
		t.Fatal("cannot change access mode to", validCache.rootDir)
	}
	_, err = validCache.OciTempImage(validSHASum, validPath)
	if err == nil {
		t.Fatal("OciTempImage() succeeded with a invalid cache")
	}
}

func TestOciTempExists(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	newCache := createTempCache(t)
	if newCache == nil {
		t.Fatal("failed to create temporary cache")
	}
	defer newCache.Clean()

	// We create a file in the cache to simulate a valid image
	createFakeImage(t, newCache.OciTemp)

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
			expected: false,
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
			exists, err := newCache.OciTempExists(test.sum, test.path)
			if test.expected == true && err != nil {
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
