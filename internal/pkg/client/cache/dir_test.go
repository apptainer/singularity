// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cache

import (
	"path/filepath"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
	"github.com/sylabs/singularity/pkg/syfs"
)

const cacheCustom = "/tmp/customcachedir"

var expectedCacheCustomRoot = filepath.Join(cacheCustom, CacheDir)
var cacheDefault = filepath.Join(syfs.ConfigDir(), CacheDir)

func TestRoot(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	tests := []struct {
		name     string
		dir      string
		expected string
	}{
		{
			name:     "Default root",
			dir:      "",
			expected: cacheDefault,
		},
		{
			name:     "Custom root",
			dir:      cacheCustom,
			expected: expectedCacheCustomRoot,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewHandle(tt.dir)
			if c == nil || err != nil {
				t.Fatal("failed to create new image cache handle")
			}
			/* This is evil: if the cache is the default cache, we clean it */
			defer c.cleanAllCaches()

			if r := c.rootDir; r != tt.expected {
				t.Errorf("Unexpected result: %s (expected %s)", r, tt.expected)
			}
		})
	}
}
