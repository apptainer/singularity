// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package build

import (
    "testing"
)

// TestChef sees if we can build a SIF image from a docker://ubuntu:18.04 based kitchen to /tmp
func TestChef(t *testing.T) {
    dpf := &DockerPullFurnisher{}

    if err := dpf.Pull("//ubuntu:18.04"); err !=nil {
        t.Fatal("failed to pull:", err)
    }

    k, err := dpf.Furnish()

    if err != nil {
        t.Fatal("failed to furnish:", err)
    }

    c := &SIFChef{}

    err = c.Cook(k, "/tmp/docker_chef_test.sif")
    if err != nil {
        t.Fatal("failed to cook:", err)
    }
}
