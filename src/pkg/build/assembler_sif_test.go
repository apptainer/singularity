// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"testing"
)

// TestChef sees if we can build a SIF image from a docker://ubuntu:18.04 based kitchen to /tmp
func TestAssembler(t *testing.T) {
	dcp := &DockerConveyorPacker{}

	if err := dcp.Get("//ubuntu:18.04"); err != nil {
		t.Fatal("Conveyor failed to get:", err)
	}

	k, err := dcp.Pack()

	if err != nil {
		t.Fatal("Packer failed to pack:", err)
	}

	a := &SIFAssembler{}

	err = a.Assemble(k, "/tmp/docker_assemble_test.sif")
	if err != nil {
		t.Fatal("Assembler failed to assemble:", err)
	}
}
