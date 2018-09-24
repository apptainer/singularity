// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"os"
	"os/exec"
	"testing"

	"github.com/singularityware/singularity/src/pkg/test"
)

func buildImage(definition string, image string) ([]byte, error) {
	args := []string{"build", image, definition}
	cmd := exec.Command(cmdPath, args...)

	return cmd.CombinedOutput()
}

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

func TestInstance(t *testing.T) {
	var defaultDefinition = "testdata/instance_test/Singularity"
	var defaultImage = "Default.sif"

	buildImageOutput, buildImageError := buildImage(defaultDefinition, defaultImage)
	defer os.Remove(defaultImage)

	if buildImageError != nil {
		t.Fatalf("Error building an image from a definition: %v. Output follows.\n%s", buildImageError, string(buildImageOutput))
	}

	t.Run("StartListStop", test.WithoutPrivilege(func(t *testing.T) {
		var defaultInstance = "www"

		startInstanceOutput, startInstanceError := startInstance(defaultImage, defaultInstance)

		if startInstanceError != nil {
			t.Fatalf("Error starting instance from an image: %v. Output follows.\n%s", startInstanceError, string(startInstanceOutput))
		}

		listInstanceOutput, listInstanceError := listInstance()

		if listInstanceError != nil {
			t.Fatalf("Error listing instances: %v. Output follows.\n%s", listInstanceError, string(listInstanceOutput))
		}

		stopInstanceOutput, stopInstanceError := stopInstance(defaultInstance)

		if stopInstanceError != nil {
			t.Fatalf("Error stopping instance by name: %v. Output follows.\n%s", stopInstanceError, string(stopInstanceOutput))
		}
	}))
}
