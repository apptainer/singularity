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

func TestOciBlob(t *testing.T) {
	tests := []struct {
		name     string
		env      string
		expected string
	}{
		{"Default OCI blob", "", filepath.Join(cacheDefault, "oci")},
		{"Custom OCI blob", cacheCustom, filepath.Join(cacheCustom, "oci")},
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
	tests := []struct {
		name     string
		env      string
		expected string
	}{
		{"Default OCI temp", "", filepath.Join(cacheDefault, "oci-tmp")},
		{"Custom OCI temp", cacheCustom, filepath.Join(cacheCustom, "oci-tmp")},
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
	tests := []struct {
		name     string
		sum      string
		path     string
		expected bool
	}{
		{"empty", "", "", true},
		{"invalid", "not a SHA sum", "not an image", false},
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
