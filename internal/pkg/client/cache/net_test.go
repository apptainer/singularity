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

func TestNet(t *testing.T) {
	tests := []struct {
		name     string
		env      string
		expected string
	}{
		{"Default Net", "", filepath.Join(cacheDefault, "net")},
		{"Custom Net", cacheCustom, filepath.Join(cacheCustom, "net")},
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
