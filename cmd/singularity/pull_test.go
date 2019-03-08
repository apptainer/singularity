// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"os"
	"os/exec"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
)

func imagePull(library string, imagePath string, sourceSpec string, force, unauthenticated bool) ([]byte, error) {
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
	if imagePath != "" {
		argv = append(argv, imagePath)
	}
	argv = append(argv, sourceSpec)

	return exec.Command(cmdPath, argv...).CombinedOutput()
}

func TestPull(t *testing.T) {
	test.DropPrivilege(t)

	imagePath := "./test_pull.sif"

	tests := []struct {
		name            string
		sourceSpec      string
		force           bool
		unauthenticated bool
		library         string
		imagePath       string
		success         bool
	}{
		{"Pull_Library", "library://alpine:3.8", false, false, "", imagePath, true}, // https://cloud.sylabs.io/library
		{"Force", "library://alpine:3.8", true, false, "", imagePath, true},
		{"Unsigned_image", "library://sylabs/tests/unsigned:1.0.0", true, true, "", imagePath, true},
		{"Pull_Docker", "docker://alpine:3.8", true, false, "", imagePath, true},   // https://hub.docker.com/
		{"Pull_Shub", "shub://GodloveD/busybox", true, false, "", imagePath, true}, // https://singularity-hub.org/
		{"PullWithHash", "library://sylabs/tests/signed:sha256.5c439fd262095766693dae95fb81334c3a02a7f0e4dc6291e0648ed4ddc61c6c", true, false, "", imagePath, true},
		{"PullWithoutTransportProtocol", "alpine:3.8", true, false, "", imagePath, true},
	}
	defer os.Remove(imagePath)
	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			if b, err := imagePull(tt.library, tt.imagePath, tt.sourceSpec, tt.force, tt.unauthenticated); err != nil {
				t.Log(string(b))
				t.Fatalf("unexpected failure: %v", err)
			}
			imageVerify(t, tt.imagePath, false)
		}))
	}
}

func TestPullNonExistent(t *testing.T) {
	test.DropPrivilege(t)

	if b, err := imagePull("", "", "library://this_should_not/exist", false, false); err == nil {
		t.Log(string(b))
		t.Fatalf("unexpected success")
	}
}

func TestUnauthenticatedPull(t *testing.T) {
	test.DropPrivilege(t)

	// pull a unsigned container. should fail
	if b, err := imagePull("", imagePath, "library://sylabs/tests/unsigned:1.0.0", true, false); err == nil {
		t.Log(string(b))
		t.Fatalf("unexpected success when pulling a unsigned container")
	}

	// pull a unsigned container (with a hash). should fail
	if b, err := imagePull("", imagePath, "library://sylabs/tests/unsigned:sha256.6dabf826cd9adf3f6390748176bf8a6ff7cb574d4ee1f0e449ba5d14ae35c2bb", true, false); err == nil {
		t.Log(string(b))
		t.Fatalf("unexpected success when pulling a unsigned container with hash")
	}

}
