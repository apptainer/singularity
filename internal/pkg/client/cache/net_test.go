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

func TestNetImage(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	cacheObj := createTempCache(t)
	if cacheObj == nil {
		t.Fatal("cannot create cache object")
	}
	cacheObj.ValidState = false
	defer cacheObj.Clean()

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
			expected: false,
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := cacheObj.NetImage(tt.sum, tt.path)
			if tt.expected == false && err == nil {
				t.Fatal("invalid case succeeded")
			}
			if tt.expected == true && err != nil {
				t.Fatal("valid case failed")
			}
		})
	}
}

func TestNet(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	expectedDefaultCache, expectedCustomCache := getDefaultCacheValues(t)

	tests := []struct {
		name     string
		env      string
		expected string
	}{
		{
			name:     "Default Net",
			env:      "",
			expected: filepath.Join(expectedDefaultCache, "net"),
		},
		{
			name:     "Custom Net",
			env:      cacheCustom,
			expected: filepath.Join(expectedCustomCache, "net"),
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

			if newCache.Net != tt.expected {
				t.Errorf("Unexpected result: %s (expected %s)", newCache.Net, tt.expected)
			}
		})
	}
}

func TestNetImageExists(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	newCache := createTempCache(t)
	if newCache == nil {
		t.Fatal("failed to create temporary cache")
	}
	defer newCache.Clean()

	// We create a file in the cache to simulate a valid image
	createFakeImage(t, newCache.Net)

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

	// First we run the tests with a valid cache obj
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			exists, err := newCache.NetImageExists(test.sum, test.path)
			if test.expected == true && err != nil {
				t.Fatal("NetImageExists() failed")
			}
			if test.expected == false && exists == true {
				t.Fatal("NetImageExists() did not fail for an invalid case")
			}
			if test.expected == true && exists == false {
				t.Fatal("NetImageExists() failed while expected to succeed")
			}
		})
	}

	// Then is an invalid cache object
	newCache.ValidState = false
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := newCache.NetImageExists(test.sum, test.path)
			// the test is always supposed to fail with an invalid cache
			if err == nil {
				t.Fatal("test with invalid cache succeeded")
			}
		})
	}
}
