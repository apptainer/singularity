// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cache

import (
	"os"
	"os/user"
	"path"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/sylog"
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

func TestRoot(t *testing.T) {
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
			expected: cacheCustom,
		},
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
