// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"testing"
)

const (
	dockerURI = "docker://alpine"
)

// TestPull tests if we can pull an ubuntu image from dockerhub
func TestConveyor(t *testing.T) {
	def, err := NewDefinitionFromURI(dockerURI)
	if err != nil {
		t.Fatalf("unable to parse URI %s: %v\n", dockerURI, err)
	}

	dc := &DockerConveyor{}

	if err := dc.Get(def); err != nil {
		t.Fatalf("failed to Get from %s: %v\n", dockerURI, err)
	}
}

// TestFurnish checks if we can create a Kitchen
func TestPacker(t *testing.T) {
	def, err := NewDefinitionFromURI(dockerURI)
	if err != nil {
		t.Fatalf("unable to parse URI %s: %v\n", dockerURI, err)
	}

	dcp := &DockerConveyorPacker{}

	if err := dcp.Get(def); err != nil {
		t.Fatalf("failed to Get from %s: %v\n", dockerURI, err)
	}

	_, err = dcp.Pack()

	if err != nil {
		t.Fatalf("failed to Pack from %s: %v\n", dockerURI, err)
	}
}
