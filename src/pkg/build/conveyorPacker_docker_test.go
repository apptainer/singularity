// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"testing"

	"github.com/singularityware/singularity/src/pkg/sylog"
)

// TestPull tests if we can pull an ubuntu image from dockerhub
func TestConveyor(t *testing.T) {
	def, err := NewDefinitionFromURI("docker://ubuntu:18.04")
	if err != nil {
		sylog.Fatalf("unable to parse URI docker://ubuntu:18.04: %v\n", err)
	}

	dc := &DockerConveyor{}

	if err := dc.Get(&def); err != nil {
		t.Fatal("failed to pull:", err)
	}
}

// TestFurnish checks if we can create a Kitchen
func TestPacker(t *testing.T) {
	def, err := NewDefinitionFromURI("docker://ubuntu:18.04")
	if err != nil {
		sylog.Fatalf("unable to parse URI docker://ubuntu:18.04: %v\n", err)
	}

	dcp := &DockerConveyorPacker{}

	if err := dcp.Get(&def); err != nil {
		t.Fatal("failed to pull:", err)
	}

	_, err = dcp.Pack()

	if err != nil {
		t.Fatal("failed to furnish:", err)
	}
}
