// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// This file is been deprecated and will disappear on with version 3.3
// of singularity. The functionality has been moved to e2e/pull/pull.go

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/client/cache"
	"github.com/sylabs/singularity/internal/pkg/test"
)

// pullSylabsPublicKey will pull the default Sylabs public key.
func pullSylabsPublicKey() ([]byte, error) {
	var argv []string
	argv = append(argv, "key", "pull", "8883491F4268F173C6E5DC49EDECE4F3F38D871E")
	return exec.Command(cmdPath, argv...).CombinedOutput()
}

func imagePull(t *testing.T, library, pullDir string, imagePath string, sourceSpec string, force, unauthenticated bool) ([]byte, error) {
	// Create a clean image cache
	imgCacheDir := test.SetCacheDir(t, "")
	defer test.CleanCacheDir(t, imgCacheDir)
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
		argv = append(argv, "--dir", "/tmp")
	}
	if imagePath != "" {
		argv = append(argv, imagePath)
	}
	argv = append(argv, sourceSpec)

	cmd := exec.Command(cmdPath, argv...)
	cmd.Env = append(os.Environ(), cacheEnvStr)

	return cmd.CombinedOutput()
}

func TestPull(t *testing.T) {
	test.DropPrivilege(t)

	imagePath := "./test_pull.sif"

	if b, err := pullSylabsPublicKey(); err != nil {
		t.Log(string(b))
		t.Fatalf("Unable to download default key: %v", err)
	}

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
		{"Unsigned_image_fail", "library://sylabs/tests/unsigned:1.0.0", true, false, "", "", imagePath, false}, // pull a unsigned image; should fail
		{"NotDefaultFail", "library://sylabs/tests/not-default:1.0.0", true, false, "", "", imagePath, false},   // pull a untrusted container; should fail
		{"NotDefaultFail1", "library://sylabs/tests/not-default:1.0.0", false, false, "", "/tmp", "", false},    // pull a untrusted container; should fail
		{"NotDefaultFail2", "library://sylabs/tests/not-default:1.0.0", true, false, "", "/tmp", "", false},     // pull a untrusted container; should fail
		{"NotDefaultSuc", "library://sylabs/tests/not-default:1.0.0", true, true, "", "", imagePath, true},      // pull a untrusted container with -U
		{"NotDefault1", "library://sylabs/tests/not-default:1.0.0", false, false, "", "/tmp", "", false},        // pull a untrusted container; should fail
		{"NotDefault2", "library://sylabs/tests/not-default:1.0.0", true, false, "", "", imagePath, false},      // pull a untrusted container; should fail
		{"NotDefaultPath", "library://sylabs/tests/not-default:1.0.0", true, true, "", "/tmp", imagePath, true}, // pull a untrusted container with -U, and --path <path>
		{"NotDefaultFail2", "library://sylabs/tests/not-default:1.0.0", false, false, "", "/tmp", "", false},    // pull a untrusted container; should fail
		{"Pull_Docker", "docker://alpine:3.8", true, false, "", "", imagePath, true},                            // https://hub.docker.com/
		{"Pull_Shub", "shub://GodloveD/busybox", true, false, "", "", imagePath, true},                          // https://singularity-hub.org/
		{"PullWithHash", "library://sylabs/tests/signed:sha256.5c439fd262095766693dae95fb81334c3a02a7f0e4dc6291e0648ed4ddc61c6c", true, true, "", "", imagePath, true},
		{"PullWithoutTransportProtocol", "alpine:3.8", true, true, "", "", imagePath, true},
		{"PullNonExistent", "library://this_should_not/exist/not_exist", true, false, "", "", imagePath, false}, // pull a non-existent container
		{"Pull_Library_Latest", "library://alpine:latest", true, true, "", "", imagePath, true},                 // https://cloud.sylabs.io/library
		{"Pull_Library_Latest", "library://alpine:latest", true, true, "", "", imagePath, true},                 // https://cloud.sylabs.io/library
		{"Pull_Dir_name", "library://alpine:3.9", true, true, "", "/tmp", imagePath, true},                      // Pull the image to /tmp/test_pull.sif
		{"PullDirNameFail", "library://alpine:3.9", false, true, "", "/tmp", imagePath, false},                  // Pull the image to /tmp/test_pull.sif
		{"PullDirNameFail1", "library://alpine:3.9", false, false, "", "/tmp", imagePath, false},                // Pull the image to /tmp/test_pull.sif
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
