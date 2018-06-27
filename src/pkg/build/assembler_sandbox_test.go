// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the URIs of this project regarding your
// rights to use or distribute this software.

package build

import (
	"os"
	"testing"
)

const (
	assemblerDockerDestDir = "/tmp/docker_alpine_assemble_test"
	assemblerShubDestDir   = "/tmp/shub_alpine_assemble_test"
)

// TestAssembler sees if we can build a SIF image from a docke based kitchen to /tmp
func TestSandboxAssemblerDocker(t *testing.T) {
	def, err := NewDefinitionFromURI(assemblerDockerURI)
	if err != nil {
		t.Fatalf("unable to parse URI %s: %v\n", assemblerDockerURI, err)
	}

	ocp := &OCIConveyorPacker{}

	if err := ocp.Get(def); err != nil {
		t.Fatalf("failed to Get from %s: %v\n", assemblerDockerURI, err)
	}

	b, err := ocp.Pack()
	if err != nil {
		t.Fatalf("failed to Pack from %s: %v\n", assemblerDockerURI, err)
	}

	a := &SandboxAssembler{}

	err = a.Assemble(b, assemblerDockerDestDir)
	if err != nil {
		t.Fatalf("failed to assemble from %s: %v\n", assemblerDockerURI, err)
	}

	defer os.RemoveAll(assemblerDockerDestDir)
}
func TestSandboxAssemblerShub(t *testing.T) {
	def, err := NewDefinitionFromURI(assemblerShubURI)
	if err != nil {
		t.Fatalf("unable to parse URI %s: %v\n", assemblerShubURI, err)
	}

	scp := &ShubConveyorPacker{}

	if err := scp.Get(def); err != nil {
		t.Fatalf("failed to Get from %s: %v\n", assemblerShubURI, err)
	}

	b, err := scp.Pack()
	if err != nil {
		t.Fatalf("failed to Pack from %s: %v\n", assemblerShubURI, err)
	}

	a := &SIFAssembler{}

	err = a.Assemble(b, assemblerShubDestDir)
	if err != nil {
		t.Fatalf("failed to assemble from %s: %v\n", assemblerShubURI, err)
	}

	defer os.RemoveAll(assemblerShubDestDir)
}
