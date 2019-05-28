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
			expected: filepath.Join(cacheCustom, "net"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer Clean()
			defer os.Unsetenv(DirEnv)

			os.Setenv(DirEnv, tt.env)

			if r := Net(); r != tt.expected {
				t.Errorf("Unexpected result: %s (expected %s)", r, tt.expected)
			}
		})
	}
}

func TestNetImageExists(t *testing.T) {
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
			name:     "valid data",
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
			name:     "invalid data",
			sum:      invalidSHASum,
			path:     invalidPath,
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			exists, err := NetImageExists(test.sum, test.path)
			if err != nil {
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
}
