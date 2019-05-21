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
	"github.com/sylabs/singularity/pkg/syfs"
)

var cacheDefault = filepath.Join(syfs.ConfigDir(), CacheDir)

const cacheCustom = "/tmp/customcachedir"

func TestRoot(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	tests := []struct {
		name     string
		env      string
		expected string
	}{
		{
			name:     "Default root",
			env:      "",
			expected: cacheDefault,
		},
		{
			name:     "Custom root",
			env:      cacheCustom,
			expected: filepath.Join(cacheCustom, CacheDir),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(DirEnv, tt.env)
			defer os.Unsetenv(DirEnv)

			// This test is using the default cache, do not clean it
			c, err := NewHandle()
			if c == nil || err != nil {
				t.Fatal("failed to create cache handle")
			}

			if c.rootDir != tt.expected {
				t.Errorf("Unexpected result: %s (expected %s)", c.rootDir, tt.expected)
			}
		})
	}
}
