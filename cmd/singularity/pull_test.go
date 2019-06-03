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

	"github.com/sylabs/singularity/internal/pkg/test"
)

func imagePull(library, pullDir string, imagePath string, sourceSpec string, force, unauthenticated bool) ([]byte, error) {
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

	return exec.Command(cmdPath, argv...).CombinedOutput()
}

// tmpDirReturn will return a tmp dir path in /tmp.
func tmpDirReturn(t *testing.T) string {
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
		{"Pull_Library", "library://alpine:3.8", false, true, "", "", imagePath, true}, // pull --allow-unauthenticated ./test_pull.sif library://alpine:3.8
		{"ForceAuth", "library://alpine:3.8", true, true, "", "", imagePath, true},     // pull --force --allow-unauthenticated ./test_pull.sif library://alpine:3.8
		{"Force", "library://alpine:3.8", true, false, "", "", imagePath, true},        // pull --force ./test_pull.sif library://alpine:3.8
		{"ForceUnauth", "library://sylabs/tests/unsigned:1.0.0", true, false, "", "", imagePath, false},
		{"Unsigned_image", "library://sylabs/tests/unsigned:1.0.0", true, true, "", "", imagePath, true},
		{"Unsigned_image_fail", "library://sylabs/tests/unsigned:1.0.0", true, false, "", "", imagePath, false},
		{"NotDefault", "library://sylabs/tests/not-default:1.0.0", true, false, "", "", imagePath, true},
		{"NotDefaultU", "library://sylabs/tests/not-default:1.0.0", true, true, "", "", imagePath, true},
		{"NotDefaultSuc", "library://sylabs/tests/not-default:1.0.0", true, true, "", "", imagePath, true},
		{"NotDefault1", "library://sylabs/tests/not-default:1.0.0", false, false, "", tmpDirReturn(t), imagePath, true},
		{"NotDefault2", "library://sylabs/tests/not-default:1.0.0", true, false, "", "", imagePath, true},
		{"NotDefaultPath", "library://sylabs/tests/not-default:1.0.0", true, true, "", tmpDirReturn(t), imagePath, true},
		{"NotDefaultFail2", "library://sylabs/tests/not-default:1.0.0", false, false, "", "/tmp", "", false},
		{"Pull_Docker", "docker://alpine:3.8", true, false, "", "", imagePath, true},   // https://hub.docker.com/
		{"Pull_Shub", "shub://GodloveD/busybox", true, false, "", "", imagePath, true}, // https://singularity-hub.org/
		{"PullWithHash", "library://sylabs/tests/signed:sha256.5c439fd262095766693dae95fb81334c3a02a7f0e4dc6291e0648ed4ddc61c6c", true, true, "", "", imagePath, true},
		{"PullWithoutTransportProtocol", "alpine:3.8", true, true, "", "", imagePath, true},
		{"PullNonExistent", "library://this_should_not/exist/not_exist", true, false, "", "", imagePath, false}, // pull a non-existent container
		{"Pull_Library_Latest", "library://alpine:latest", true, true, "", "", imagePath, true},                 // https://cloud.sylabs.io/library
		{"Pull_Library_Latest", "library://alpine:latest", true, true, "", "", imagePath, true},                 // https://cloud.sylabs.io/library
		{"Pull_Dir_name", "library://alpine:3.9", true, true, "", tmpDirReturn(t), imagePath, true},             // Pull the image to /tmp/test_pull.sif
		{"PullDirNameFail", "library://alpine:3.9", false, true, "", "/tmp", imagePath, false},                  // Pull the image to /tmp/test_pull.sif
		{"PullDirNameFail1", "library://alpine:3.9", false, false, "", "/tmp", imagePath, false},                // Pull the image to /tmp/test_pull.sif
	}
	defer os.Remove(imagePath)
	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			b, err := imagePull(tt.library, tt.pullDir, tt.imagePath, tt.sourceSpec, tt.force, tt.unauthenticated)
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
