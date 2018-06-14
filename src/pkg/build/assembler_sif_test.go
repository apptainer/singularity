// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"testing"

	"github.com/singularityware/singularity/src/pkg/sylog"
)

// TestChef sees if we can build a SIF image from a docker://ubuntu:18.04 based kitchen to /tmp
func TestAssembler(t *testing.T) {
	def, err := NewDefinitionFromURI("docker://ubuntu:18.04")
	if err != nil {
		sylog.Fatalf("unable to parse URI docker://ubuntu:18.04: %v\n", err)
	}

	dcp := &DockerConveyorPacker{}

	if err := dcp.Get(def); err != nil {
		t.Fatal("failed to pull:", err)
	}

	b, err := dcp.Pack()

	if err != nil {
		t.Fatal("failed to furnish:", err)
	}

	a := &SIFAssembler{}

	err = a.Assemble(b, "/tmp/docker_assemble_test.sif")
	if err != nil {
		t.Fatal("Assembler failed to assemble:", err)
	}
}
