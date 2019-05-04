// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cache

import (
	"os"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
)

func TestRoot(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	expectedDefaultRoot, expectedCustomRoot := getDefaultCacheValues(t)

	tests := []struct {
		name     string
		env      string
		expected string
	}{
		{
			name:     "Default root",
			env:      "",
			expected: expectedDefaultRoot,
		},
		{
			name:     "Custom root",
			env:      cacheCustom,
			expected: expectedCustomRoot,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(DirEnv, tt.env)
			defer os.Unsetenv(DirEnv)

			newCache := createTempCache(t)
			if newCache == nil {
				t.Fatal("failed to create temporary cache")
			}
			defer newCache.Clean()

			if r := newCache.Root; r != tt.expected {
				t.Errorf("Unexpected result: %s (expected %s)", r, tt.expected)
			}
		})
	}
}
