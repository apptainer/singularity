// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// This file is been deprecated and will disappear on with version 3.3
// of singularity. The functionality has been moved to e2e/pull/pull.go

package main

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/client/cache"
	"github.com/sylabs/singularity/internal/pkg/test"
	testCache "github.com/sylabs/singularity/internal/pkg/test/tool/cache"
)

func imagePull(t *testing.T, library, pullDir string, imagePath string, sourceSpec string, force, unauthenticated bool) ([]byte, error) {
	// Create a clean image cache
	imgCacheDir := testCache.MakeDir(t, "")
	defer testCache.DeleteDir(t, imgCacheDir)
	cacheEnvStr := cache.DirEnv + "=" + imgCacheDir

	var argv []string
	argv = append(argv, "pull")
	if force {
		argv = append(argv, "--force")
	}
	if unauthenticated {
		argv = append(argv, "--allow-unauthenticated")
	}
	if library != "" {
		argv = append(argv, "--library", library)
	}
	if pullDir != "" {
		argv = append(argv, "--dir", pullDir)
	}
	if imagePath != "" {
		argv = append(argv, imagePath)
	}
	argv = append(argv, sourceSpec)

	cmd := exec.Command(cmdPath, argv...)
	cmd.Env = append(os.Environ(), cacheEnvStr)

	return cmd.CombinedOutput()
}

// makeTmpDir will return a tmp dir path in /tmp.
func makeTmpDir(t *testing.T) string {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("Unable to make tmp dir: %v", err)
	}
	return tmpDir
}

func TestPull(t *testing.T) {
	test.DropPrivilege(t)

	imagePath := "./test_pull.sif"

	// nolint:maligned
	tests := []struct {
		name            string
		sourceSpec      string
		force           bool
		unauthenticated bool
		library         string
		pullDir         string
		imagePath       string
		success         bool
	}{
		{
			name:            "Pull_Library",
			sourceSpec:      "library://alpine:3.8",
			force:           false,
			unauthenticated: true,
			library:         "",
			pullDir:         "",
			imagePath:       imagePath,
			success:         true,
		},
		{
			name:            "ForceAuth",
			sourceSpec:      "library://alpine:3.8",
			force:           true,
			unauthenticated: true,
			library:         "",
			pullDir:         "",
			imagePath:       imagePath,
			success:         true,
		},
		{
			name:            "Force",
			sourceSpec:      "library://alpine:3.8",
			force:           true,
			unauthenticated: false,
			library:         "",
			pullDir:         "",
			imagePath:       imagePath,
			success:         true,
		},
		{
			name:            "ForceUnauth",
			sourceSpec:      "library://sylabs/tests/unsigned:1.0.0",
			force:           true,
			unauthenticated: false,
			library:         "",
			pullDir:         "",
			imagePath:       imagePath,
			success:         true,
		},
		{
			name:            "Unsigned_image",
			sourceSpec:      "library://sylabs/tests/unsigned:1.0.0",
			force:           true,
			unauthenticated: true,
			library:         "",
			pullDir:         "",
			imagePath:       imagePath,
			success:         true,
		},
		{
			name:            "NotDefault",
			sourceSpec:      "library://sylabs/tests/not-default:1.0.0",
			force:           true,
			unauthenticated: false,
			library:         "",
			pullDir:         "",
			imagePath:       imagePath,
			success:         true,
		},
		{
			name:            "NotDefaultU",
			sourceSpec:      "library://sylabs/tests/not-default:1.0.0",
			force:           true,
			unauthenticated: true,
			library:         "",
			pullDir:         "",
			imagePath:       imagePath,
			success:         true,
		},
		{
			name:            "NotDefaultSuc",
			sourceSpec:      "library://sylabs/tests/not-default:1.0.0",
			force:           true,
			unauthenticated: true,
			library:         "",
			pullDir:         "",
			imagePath:       imagePath,
			success:         true,
		},
		{
			name:            "NotDefault1",
			sourceSpec:      "library://sylabs/tests/not-default:1.0.0",
			force:           false,
			unauthenticated: false,
			library:         "",
			pullDir:         makeTmpDir(t),
			imagePath:       imagePath,
			success:         true,
		},
		{
			name:            "NotDefault2",
			sourceSpec:      "library://sylabs/tests/not-default:1.0.0",
			force:           true,
			unauthenticated: false,
			library:         "",
			pullDir:         "",
			imagePath:       imagePath,
			success:         true,
		},
		{
			name:            "NotDefaultPath",
			sourceSpec:      "library://sylabs/tests/not-default:1.0.0",
			force:           true,
			unauthenticated: true,
			library:         "",
			pullDir:         makeTmpDir(t),
			imagePath:       imagePath,
			success:         true,
		},
		{
			name:            "NotDefaultPath2",
			sourceSpec:      "library://sylabs/tests/not-default:1.0.0",
			force:           false,
			unauthenticated: false,
			library:         "",
			pullDir:         makeTmpDir(t),
			imagePath:       imagePath,
			success:         true,
		},
		{
			name:            "Pull_Docker",
			sourceSpec:      "docker://alpine:3.8",
			force:           true,
			unauthenticated: false,
			library:         "",
			pullDir:         "",
			imagePath:       imagePath,
			success:         true,
		},
		// TODO: Uncomment when shub is working
		/*		{
				name:            "Pull_Shub",
				sourceSpec:      "shub://GodloveD/busybox",
				force:           true,
				unauthenticated: false,
				library:         "",
				pullDir:         "",
				imagePath:       imagePath,
				success:         true,
			},*/
		{
			name:            "PullWithHash",
			sourceSpec:      "library://sylabs/tests/signed:sha256.5c439fd262095766693dae95fb81334c3a02a7f0e4dc6291e0648ed4ddc61c6c",
			force:           true,
			unauthenticated: true,
			library:         "",
			pullDir:         "",
			imagePath:       imagePath,
			success:         true,
		},
		{
			name:            "PullWithoutTransportProtocol",
			sourceSpec:      "alpine:3.8",
			force:           true,
			unauthenticated: true,
			library:         "",
			pullDir:         "",
			imagePath:       imagePath,
			success:         true,
		},
		{
			name:            "PullNonExistent",
			sourceSpec:      "library://sylabs/exist/not_exist",
			force:           true,
			unauthenticated: false,
			library:         "",
			pullDir:         "",
			imagePath:       imagePath,
			success:         false,
		},
		{
			name:            "Pull_Library_Latest",
			sourceSpec:      "library://alpine:latest",
			force:           true,
			unauthenticated: true,
			library:         "",
			pullDir:         "",
			imagePath:       imagePath,
			success:         true,
		},
		{
			name:            "Pull_Library_Latest",
			sourceSpec:      "library://alpine:latest",
			force:           true,
			unauthenticated: true,
			library:         "",
			pullDir:         "",
			imagePath:       imagePath,
			success:         true,
		},
		{
			name:            "Pull_Dir_name",
			sourceSpec:      "library://alpine:3.9",
			force:           true,
			unauthenticated: true,
			library:         "",
			pullDir:         "/tmp",
			imagePath:       imagePath,
			success:         true,
		},
		{
			name:            "PullDirNameFail",
			sourceSpec:      "library://alpine:3.9",
			force:           false,
			unauthenticated: true,
			library:         "",
			pullDir:         "/tmp",
			imagePath:       imagePath,
			success:         false,
		},
		{
			name:            "PullDirNameFail1",
			sourceSpec:      "library://alpine:3.9",
			force:           false,
			unauthenticated: false,
			library:         "",
			pullDir:         "/tmp",
			imagePath:       imagePath,
			success:         false,
		},
	}
	defer os.Remove(imagePath)
	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			b, err := imagePull(t, tt.library, tt.pullDir, tt.imagePath, tt.sourceSpec, tt.force, tt.unauthenticated)
			if tt.success {
				if err != nil {
					t.Log(string(b))
					t.Fatalf("unexpected failure: %v", err)
				}
				imageVerify(t, filepath.Join(tt.pullDir, tt.imagePath), false)
			} else {
				if err == nil {
					t.Log(string(b))
					t.Fatalf("unexpected success: command should have failed")
				}
			}
		}))
	}
}
