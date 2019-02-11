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

func imagePull(library string, imagePath string, sourceSpec string, force bool) ([]byte, error) {
	var argv []string
	argv = append(argv, "pull")
	if force {
		argv = append(argv, "--force")
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
		name       string
		sourceSpec string
		force      bool
		library    string
		imagePath  string
		success    bool
	}{
		{"Pull_Library", "library://alpine:3.7", false, "", imagePath, true}, // https://cloud.sylabs.io/library
		{"Force", "library://alpine:3.7", true, "", imagePath, true},
		{"Pull_Docker", "docker://alpine:3.7", true, "", imagePath, true},   // https://hub.docker.com/
		{"Pull_Shub", "shub://GodloveD/busybox", true, "", imagePath, true}, // https://singularity-hub.org/
		{"PullWithHash", "library://alpine:sha256.af5e6b93dcfd08b7b46d6158c31ae5b6c2b4241c169b65595659ad9feac6f761", true, "", imagePath, true},
		{"PullWithoutTransportProtocol", "alpine:3.7", true, "", imagePath, true},
	}
	defer os.Remove(imagePath)
	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			if b, err := imagePull(tt.library, tt.imagePath, tt.sourceSpec, tt.force); err != nil {
				t.Log(string(b))
				t.Fatalf("unexpected failure: %v", err)
			}
			imageVerify(t, tt.imagePath, false)
		}))
	}
}

func TestPullNonExistent(t *testing.T) {
	test.DropPrivilege(t)

	if b, err := imagePull("", "", "library://this_should_not/exist", false); err == nil {
		t.Log(string(b))
		t.Fatalf("unexpected success")
	}
}
