// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cache

import (
	"os"
	"os/user"
	"path"
	"path/filepath"
	"testing"

	"github.com/sylabs/singularity/src/pkg/sylog"
)

var cacheDefault string

const cacheCustom = "/tmp/customcachedir"

func TestMain(m *testing.M) {
	usr, err := user.Current()
	if err != nil {
		sylog.Errorf("Couldn't determine user home directory: %v", err)
		os.Exit(1)
	}
	cacheDefault = path.Join(usr.HomeDir, RootDefault)

	os.Exit(m.Run())
}

func Test_Root(t *testing.T) {
	tests := []struct {
		name     string
		env      string
		expected string
	}{
		{"Default root", "", cacheDefault},
		{"Custom root", cacheCustom, cacheCustom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer Clean()
			defer os.Unsetenv(DirEnv)

			os.Setenv(DirEnv, tt.env)

			if r := Root(); r != tt.expected {
				t.Errorf("Unexpected result: %s (expected %s)", r, tt.expected)
			}
		})
	}
}

func Test_OciBlob(t *testing.T) {
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

func Test_OciTemp(t *testing.T) {
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

func Test_Library(t *testing.T) {
	tests := []struct {
		name     string
		env      string
		expected string
	}{
		{"Default Library", "", filepath.Join(cacheDefault, "library")},
		{"Custom Library", "/tmp/CustomCacheDir", "/tmp/CustomCacheDir/library"},
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
