// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"os"
	"os/exec"
	"testing"

	"github.com/sylabs/singularity/src/pkg/test"
)

func startInstance(image string, instance string) ([]byte, error) {
	args := []string{"instance", "start", image, instance}
	cmd := exec.Command(cmdPath, args...)

	return cmd.CombinedOutput()
}

func listInstance() ([]byte, error) {
	args := []string{"instance", "list"}
	cmd := exec.Command(cmdPath, args...)

	return cmd.CombinedOutput()
}

func stopInstance(instance string) ([]byte, error) {
	args := []string{"instance", "stop", instance}
	cmd := exec.Command(cmdPath, args...)

	return cmd.CombinedOutput()
}

// TestInstance tests singularity instance cmd
// start, list, stop
func TestInstance(t *testing.T) {
	var definition = "../../../examples/busybox/Singularity"
	var imagePath = "./instance_tests.sif"

	opts := buildOpts{
		force:   true,
		sandbox: false,
	}
	if b, err := imageBuild(opts, imagePath, definition); err != nil {
		t.Log(string(b))
		t.Fatalf("unexpected failure: %v", err)
	}
	imageVerify(t, imagePath, true)
	defer os.RemoveAll(imagePath)

	t.Run("StartListStop", test.WithoutPrivilege(func(t *testing.T) {
		var defaultInstance = "www"

		startInstanceOutput, err := startInstance(imagePath, defaultInstance)
		if err != nil {
			t.Fatalf("Error starting instance from an image: %v. Output follows.\n%s", err, string(startInstanceOutput))
		}

		listInstanceOutput, err := listInstance()
		if err != nil {
			t.Fatalf("Error listing instances: %v. Output follows.\n%s", err, string(listInstanceOutput))
		}

		stopInstanceOutput, err := stopInstance(defaultInstance)
		if err != nil {
			t.Fatalf("Error stopping instance by name: %v. Output follows.\n%s", err, string(stopInstanceOutput))
		}
	}))
}
