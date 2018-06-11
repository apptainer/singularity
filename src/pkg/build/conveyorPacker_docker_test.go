// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"testing"
)

// TestPull tests if we can pull an ubuntu image from dockerhub
func TestConveyor(t *testing.T) {
	dc := &DockerConveyor{}

	if err := dc.Get("//ubuntu:18.04"); err != nil {
		t.Fatal("failed to pull:", err)
	}
}

// TestFurnish checks if we can create a Kitchen
func TestPacker(t *testing.T) {
	dcp := &DockerConveyorPacker{}

	if err := dcp.Get("//ubuntu:18.04"); err != nil {
		t.Fatal("failed to pull:", err)
	}

	_, err := dcp.Pack()

	if err != nil {
		t.Fatal("failed to furnish:", err)
	}
}
