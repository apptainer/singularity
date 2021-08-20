// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package bin

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"

	"github.com/hpcng/singularity/internal/pkg/buildcfg"
	"github.com/hpcng/singularity/pkg/util/singularityconf"
)

func TestFindOnPath(t *testing.T) {
	// findOnPath should give same as exec.LookPath, but additionally work
	// in the case where $PATH doesn't include default sensible directories
	// as these are added to $PATH before the lookup.
	truePath, err := exec.LookPath("cp")
	if err != nil {
		t.Fatalf("exec.LookPath failed to find cp: %v", err)
	}

	t.Run("unmodified path", func(t *testing.T) {
		gotPath, err := findOnPath("cp")
		if err != nil {
			t.Errorf("unexpected error from findOnPath: %v", err)
		}
		if gotPath != truePath {
			t.Errorf("Got %q, expected %q", gotPath, truePath)
		}
	})

	t.Run("modified path", func(t *testing.T) {
		oldPath := os.Getenv("PATH")
		defer os.Setenv("PATH", oldPath)
		os.Setenv("PATH", "/invalid/dir:/another/invalid/dir")

		gotPath, err := findOnPath("cp")
		if err != nil {
			t.Errorf("unexpected error from findOnPath: %v", err)
		}
		if gotPath != truePath {
			t.Errorf("Got %q, expected %q", gotPath, truePath)
		}
	})
}

func TestFindFromConfig(t *testing.T) {
	cases := []struct {
		name          string
		bin           string
		buildcfg      string
		expectSuccess bool
		configKey     string
		configVal     string
		expectPath    string
	}{
		{
			name:          "cryptsetup valid",
			bin:           "cryptsetup",
			buildcfg:      buildcfg.CRYPTSETUP_PATH,
			configKey:     "cryptsetup path",
			configVal:     buildcfg.CRYPTSETUP_PATH,
			expectPath:    buildcfg.CRYPTSETUP_PATH,
			expectSuccess: true,
		},
		{
			name:          "cryptsetup invalid",
			bin:           "cryptsetup",
			buildcfg:      buildcfg.CRYPTSETUP_PATH,
			configKey:     "cryptsetup path",
			configVal:     "/invalid/dir/cryptsetup",
			expectSuccess: false,
		},
		{
			name:          "cryptsetup empty",
			bin:           "cryptsetup",
			buildcfg:      buildcfg.CRYPTSETUP_PATH,
			configKey:     "cryptsetup path",
			configVal:     "",
			expectPath:    "_LOOKPATH_",
			expectSuccess: true,
		},
		{
			name:          "go valid",
			bin:           "go",
			buildcfg:      buildcfg.GO_PATH,
			configKey:     "go path",
			configVal:     buildcfg.GO_PATH,
			expectPath:    buildcfg.GO_PATH,
			expectSuccess: true,
		},
		{
			name:          "go invalid",
			bin:           "go",
			buildcfg:      buildcfg.GO_PATH,
			configKey:     "go path",
			configVal:     "/invalid/dir/go",
			expectSuccess: false,
		},
		{
			name:          "go empty",
			bin:           "go",
			buildcfg:      buildcfg.GO_PATH,
			configKey:     "go path",
			configVal:     "",
			expectPath:    "_LOOKPATH_",
			expectSuccess: true,
		},
		{
			name:          "ldconfig valid",
			bin:           "ldconfig",
			buildcfg:      buildcfg.LDCONFIG_PATH,
			configKey:     "ldconfig path",
			configVal:     buildcfg.LDCONFIG_PATH,
			expectPath:    buildcfg.LDCONFIG_PATH,
			expectSuccess: true,
		},
		{
			name:          "ldconfig invalid",
			bin:           "ldconfig",
			buildcfg:      buildcfg.LDCONFIG_PATH,
			configKey:     "ldconfig path",
			configVal:     "/invalid/dir/go",
			expectSuccess: false,
		},
		{
			name:          "ldconfig empty",
			bin:           "ldconfig",
			buildcfg:      buildcfg.LDCONFIG_PATH,
			configKey:     "ldconfig path",
			configVal:     "",
			expectPath:    "_LOOKPATH_",
			expectSuccess: true,
		},
		{
			name:          "mksquashfs valid",
			bin:           "mksquashfs",
			buildcfg:      buildcfg.MKSQUASHFS_PATH,
			configKey:     "mksquashfs path",
			configVal:     buildcfg.MKSQUASHFS_PATH,
			expectPath:    buildcfg.MKSQUASHFS_PATH,
			expectSuccess: true,
		},
		{
			name:          "mksquashfs invalid",
			bin:           "mksquashfs",
			buildcfg:      buildcfg.MKSQUASHFS_PATH,
			configKey:     "mksquashfs path",
			configVal:     "/invalid/dir/go",
			expectSuccess: false,
		},
		{
			name:          "mksquashfs empty",
			bin:           "mksquashfs",
			buildcfg:      buildcfg.MKSQUASHFS_PATH,
			configKey:     "mksquashfs path",
			configVal:     "",
			expectPath:    "_LOOKPATH_",
			expectSuccess: true,
		},
		{
			name:          "nvidia-container-cli valid",
			bin:           "nvidia-container-cli",
			buildcfg:      buildcfg.NVIDIA_CONTAINER_CLI_PATH,
			configKey:     "nvidia-container-cli path",
			configVal:     buildcfg.NVIDIA_CONTAINER_CLI_PATH,
			expectPath:    buildcfg.NVIDIA_CONTAINER_CLI_PATH,
			expectSuccess: true,
		},
		{
			name:          "nvidia-container-cli invalid",
			bin:           "nvidia-container-cli",
			buildcfg:      buildcfg.NVIDIA_CONTAINER_CLI_PATH,
			configKey:     "nvidia-container-cli path",
			configVal:     "/invalid/dir/go",
			expectSuccess: false,
		},
		{
			name:          "nvidia-container-cli empty",
			bin:           "nvidia-container-cli",
			buildcfg:      buildcfg.NVIDIA_CONTAINER_CLI_PATH,
			configKey:     "nvidia-container-cli path",
			configVal:     "",
			expectPath:    "_LOOKPATH_",
			expectSuccess: true,
		},
		{
			name:          "unsquashfs valid",
			bin:           "unsquashfs",
			buildcfg:      buildcfg.UNSQUASHFS_PATH,
			configKey:     "unsquashfs path",
			configVal:     buildcfg.UNSQUASHFS_PATH,
			expectPath:    buildcfg.UNSQUASHFS_PATH,
			expectSuccess: true,
		},
		{
			name:          "unsquashfs invalid",
			bin:           "unsquashfs",
			buildcfg:      buildcfg.UNSQUASHFS_PATH,
			configKey:     "unsquashfs path",
			configVal:     "/invalid/dir/go",
			expectSuccess: false,
		},
		{
			name:          "unsquashfs empty",
			bin:           "unsquashfs",
			buildcfg:      buildcfg.UNSQUASHFS_PATH,
			configKey:     "unsquashfs path",
			configVal:     "",
			expectPath:    "_LOOKPATH_",
			expectSuccess: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.buildcfg == "" {
				t.Skip("skipping - no buildcfg path known")
			}
			lookPath, err := exec.LookPath(tc.bin)
			if err != nil {
				t.Skipf("Error from exec.LookPath for %q: %v", tc.bin, err)
			}

			if tc.expectPath == "_LOOKPATH_" {
				tc.expectPath = lookPath
			}

			f, err := ioutil.TempFile("", "test.conf")
			if err != nil {
				t.Fatalf("cannot create temporary test configuration: %+v", err)
			}
			f.Close()
			defer os.Remove(f.Name())

			cfg := fmt.Sprintf("%s = %s\n", tc.configKey, tc.configVal)
			ioutil.WriteFile(f.Name(), []byte(cfg), 0o644)

			conf, err := singularityconf.Parse(f.Name())
			if err != nil {
				t.Errorf("Error parsing test singularityconf: %v", err)
			}
			singularityconf.SetCurrentConfig(conf)

			path, err := findFromConfig(tc.bin)

			if tc.expectSuccess && err == nil {
				// expect success, no error, check path
				if path != tc.expectPath {
					t.Errorf("Expecting %q, got %q", tc.expectPath, path)
				}
			}

			if tc.expectSuccess && err != nil {
				// expect success, got error
				t.Errorf("unexpected error: %v", err)
			}

			if !tc.expectSuccess && err == nil {
				// expect failure, got no error
				t.Errorf("expected error, got %q", path)
			}
		})
	}
}
