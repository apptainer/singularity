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

func TestLibrary(t *testing.T) {
	tests := []struct {
		name     string
		env      string
		expected string
	}{
		{"Default Library", "", filepath.Join(cacheDefault, "library")},
		{"Custom Library", cacheCustom, filepath.Join(cacheCustom, "library")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer Clean()
			defer os.Unsetenv(DirEnv)

			os.Setenv(DirEnv, tt.env)

			if r := Library(); r != tt.expected {
				t.Errorf("Unexpected result: %s (expected %s)", r, tt.expected)
			}
		})
	}
}

// Will come back to after functionality exists
// func TestLibraryImageExists(t *testing.T) {
// 	tests := []struct {
// 		name     string
// 		env      string
// 		expected string
// 		file     []struct {
// 			create bool
// 			sum    string
// 			name   string
// 		}
// 	}{}
// }
