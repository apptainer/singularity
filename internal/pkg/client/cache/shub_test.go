// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
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

func TestShubImageExists(t *testing.T) {
	tests := []struct {
		name     string
		sum      string
		path     string
		expected bool
	}{
		{"empty data", "", "", true},
		{"invalid data", "not a SHA sum", "not an image", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			exists, err := ShubImageExists(test.sum, test.path)
			if err != nil {
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
