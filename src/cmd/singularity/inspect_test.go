// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"os/exec"
	"testing"

	"github.com/sylabs/singularity/src/pkg/test"
)

func inspectImage(image string) ([]byte, error) {
	args := []string{"inspect", image}
	cmd := exec.Command(cmdPath, args...)

	return cmd.CombinedOutput()
}

func TestInspect(t *testing.T) {
	var definition = "../../../examples/busybox/Singularity"
	var image = "Default.sif"

	if b, err := buildImage(definition, image); err != nil {
		t.Fatalf("Error building an image from a definition: %v. Output follows.\n%s", err, string(b))
	}

	t.Run("InspectImage", test.WithoutPrivilege(func(t *testing.T) {
		if b, err := inspectImage(image); err != nil {
			t.Fatalf("Error inspecting an image: %v. Output follows.\n%s", err, string(b))
		}
	}))
}
