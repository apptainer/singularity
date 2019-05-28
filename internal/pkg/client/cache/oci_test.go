// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cache

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/client/cache"
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
			expected: filepath.Join(cacheCustom, "oci"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer Clean()
			defer os.Unsetenv(DirEnv)

			os.Setenv(DirEnv, tt.env)

			if r := OciBlob(); r != tt.expected {
				t.Errorf("Unexpected result: %s (expected %s)", r, tt.expected)
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
			expected: filepath.Join(cacheCustom, "oci-tmp"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer Clean()
			defer os.Unsetenv(DirEnv)

			os.Setenv(DirEnv, tt.env)

			if r := OciTemp(); r != tt.expected {
				t.Errorf("Unexpected result: %s (expected %s)", r, tt.expected)
			}
		})
	}
}

func TestOciTempExists(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	cacheDir := test.SetCacheDir(t, "")
	defer test.CleanCacheDir(t, cacheDir)

	err := os.Setenv(cache.DirEnv, cacheDir)
	if err != nil {
		t.Fatalf("failed to set %s environment variable: %s", cache.DirEnv, err)
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
			exists, err := OciTempExists(test.sum, test.path)
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
