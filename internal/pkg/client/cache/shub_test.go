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

func TestShub(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	expectedDefaultCache, expectedCustomCache := getDefaultCacheValues(t)

	tests := []struct {
		name     string
		env      string
		expected string
	}{
		{
			name:     "Default Shub",
			env:      "",
			expected: filepath.Join(expectedDefaultCache, "shub"),
		},
		{
			name:     "Custom Shub",
			env:      cacheCustom,
			expected: filepath.Join(expectedCustomCache, "shub"),
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

			if newCache.Shub != tt.expected {
				t.Errorf("Unexpected result: %s (expected %s)", newCache.Shub, tt.expected)
			}
		})
	}
}

// Only tests a few corner cases, most of the testing is done while testing ShubImageExists()
func TestShubImage(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	newCache := createTempCache(t)
	if newCache == nil {
		t.Fatal("failed to create temporary cache")
	}
	defer newCache.Clean()

	// First test, we change the access mode of the cache's root to reach a specific error case
	err := os.Chmod(newCache.rootDir, 0444)
	if err != nil {
		t.Fatal("cannot change access mode to", newCache.rootDir)
	}
	_, err = newCache.ShubImage(validSHASum, validPath)
	if err == nil {
		t.Fatal("ShubImage() succeeded while expected to fail")
	}

	// Second, we test with an cache that has an invalid state
	newCache.State = StateInvalid
	_, err = newCache.ShubImage(validSHASum, validPath)
	if err == nil {
		t.Fatalf("ShubImage() succeeded with an invalid cache")
	}
}

func TestShubImageExists(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	newCache := createTempCache(t)
	if newCache == nil {
		t.Fatal("failed to create temporary cache")
	}
	defer newCache.Clean()

	// We create a file in the cache to simulate a valid image
	createFakeImage(t, newCache.Shub)

	tests := []struct {
		name     string
		sum      string
		path     string
		expected bool
	}{
		{
			name:     "valid data",
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
			name:     "invalid data",
			sum:      invalidSHASum,
			path:     invalidPath,
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			exists, err := newCache.ShubImageExists(test.sum, test.path)
			if test.expected == true && err != nil {
				t.Fatal("ShubImageExists() failed")
			}
			if test.expected == false && exists == true {
				t.Fatal("NetImageExists() did not fail for an invalid case")
			}
			if test.expected == true && exists == false {
				t.Fatal("NetImageExists() failed while expected to succeed")
			}
		})
	}
}
