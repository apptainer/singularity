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
	assemblerDockerURI  = "docker://alpine"
	assemblerDockerDest = "/tmp/docker_alpine_assemble_test.sif"
	assemblerShubURI    = "shub://truatpasteurdotfr/singularity-alpine"
	assemblerShubDest   = "/tmp/shub_alpine_assemble_test.sif"
)

// TestAssembler sees if we can build a SIF image from a docke based kitchen to /tmp
func TestAssemblerDocker(t *testing.T) {
	def, err := NewDefinitionFromURI(assemblerDockerURI)
	if err != nil {
		t.Fatalf("unable to parse URI %s: %v\n", assemblerDockerURI, err)
	}

	dcp := &DockerConveyorPacker{}

	if err := dcp.Get(def); err != nil {
		t.Fatalf("failed to Get from %s: %v\n", assemblerDockerURI, err)
	}

	b, err := dcp.Pack()
	if err != nil {
		t.Fatalf("failed to Pack from %s: %v\n", assemblerDockerURI, err)
	}

	a := &SIFAssembler{}

	err = a.Assemble(b, assemblerDockerDest)
	if err != nil {
		t.Fatalf("failed to assemble from %s: %v\n", assemblerDockerURI, err)
	}

	defer os.Remove(assemblerDockerDest)
}
func TestAssemblerShub(t *testing.T) {
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

	err = a.Assemble(b, assemblerShubDest)
	if err != nil {
		t.Fatalf("failed to assemble from %s: %v\n", assemblerShubURI, err)
	}

	defer os.Remove(assemblerShubDest)
}
