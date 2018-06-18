// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
	"testing"
)

const (
	shubURI = "shub://singularityhub/ubuntu"
)

// TestPull tests if we can pull an ubuntu image from dockerhub
func TestShubConveyor(t *testing.T) {
	def, err := NewDefinitionFromURI(shubURI)
	if err != nil {
		t.Fatalf("unable to parse URI %s: %v\n", shubURI, err)
	}

	scp := &ShubConveyorPacker{}

	if err := scp.Get(def); err != nil {
		t.Fatalf("failed to Get from %s: %v\n", shubURI, err)
	}

	b, err := scp.Pack()
	if err != nil {
		t.Fatalf("failed to Pack from %s: %v\n", shubURI, err)
	}

	t.Fatalf(b.Rootfs())

}

// // TestFurnish checks if we can create a Kitchen
// func TestShubPacker(t *testing.T) {
// 	def, err := NewDefinitionFromURI("docker://ubuntu:18.04")
// 	if err != nil {
// 		t.Fatalf("unable to parse URI docker://ubuntu:18.04: %v\n", err)
// 	}

// 	dcp := &DockerConveyorPacker{}

// 	if err := dcp.Get( def); err != nil {
// 		t.Fatal("failed to pull:", err)
// 	}

// 	_, err = dcp.Pack()

// 	if err != nil {
// 		t.Fatal("failed to furnish:", err)
// 	}
// }
