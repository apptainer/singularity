// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cache

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sylabs/singularity/src/pkg/test"
)

func Test_Root(t *testing.T) {
	tests := []struct {
		name     string
		env      string
		expected string
	}{
		{"Default root", test.CacheDirUnpriv, test.CacheDirUnpriv},
		{"Custom root", "/tmp/CustomCacheDir", "/tmp/CustomCacheDir"},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			defer Clean()
			os.Setenv("SINGULARITY_CACHEDIR", tt.env)

			if r := Root(); r != tt.expected {
				t.Errorf("Unexpected result: %s (expected %s)", r, tt.expected)
			}
		}))
	}
}

func Test_OciBlob(t *testing.T) {
	tests := []struct {
		name     string
		env      string
		expected string
	}{
		{"Default OCI blob", test.CacheDirUnpriv, filepath.Join(test.CacheDirUnpriv, "oci")},
		{"Custom OCI blob", "/tmp/CustomCacheDir", "/tmp/CustomCacheDir/oci"},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			defer Clean()
			os.Setenv("SINGULARITY_CACHEDIR", tt.env)

			if r := OciBlob(); r != tt.expected {
				t.Errorf("Unexpected result: %s (expected %s)", r, tt.expected)
			}
		}))
	}
}

func Test_OciTemp(t *testing.T) {
	tests := []struct {
		name     string
		env      string
		expected string
	}{
		{"Default OCI temp", test.CacheDirUnpriv, filepath.Join(test.CacheDirUnpriv, "oci-tmp")},
		{"Custom OCI temp", "/tmp/CustomCacheDir", "/tmp/CustomCacheDir/oci-tmp"},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			defer Clean()
			os.Setenv("SINGULARITY_CACHEDIR", tt.env)

			if r := OciTemp(); r != tt.expected {
				t.Errorf("Unexpected result: %s (expected %s)", r, tt.expected)
			}
		}))
	}
}
