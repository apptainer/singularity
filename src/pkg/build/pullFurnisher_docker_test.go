// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
    "testing"
)

// TestPull tests if we can pull an ubuntu image from dockerhub
func TestPull(t *testing.T) {
    dp := &DockerPuller{}

    if err := dp.Pull("//ubuntu:18.04"); err !=nil {
        t.Fatal("failed to pull:", err)
    }
}

// TestFurnish checks if we can create a Kitchen
func TestFurnish(t *testing.T) {
    dpf := &DockerPullFurnisher{}

    if err := dpf.Pull("//ubuntu:18.04"); err !=nil {
        t.Fatal("failed to pull:", err)
    }

    _, err := dpf.Furnish()

    if err !=nil {
        t.Fatal("failed to furnish:", err)
    }
}
