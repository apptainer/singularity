// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cache

import (
	"os"
	"path/filepath"
	"testing"
)

func TestShub(t *testing.T) {
	tests := []struct {
		name     string
		env      string
		expected string
	}{
		{"Default Shub", "", filepath.Join(cacheDefault, "shub")},
		{"Custom Shub", cacheCustom, filepath.Join(cacheCustom, "shub")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer Clean()
			defer os.Unsetenv(DirEnv)

			os.Setenv(DirEnv, tt.env)

			if r := Shub(); r != tt.expected {
				t.Errorf("Unexpected result: %s (expected %s)", r, tt.expected)
			}
		})
	}
}
