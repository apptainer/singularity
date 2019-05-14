// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package e2e

import (
	"os/exec"
	"strings"
	"testing"
)

func PrepRegistry(t *testing.T) {
	commands := [][]string{
		{"run", "-d", "-p", "5000:5000", "--restart=always", "--name", "registry", "registry:2"},
		{"pull", "busybox"},
		{"tag", "busybox", "localhost:5000/my-busybox"},
		{"push", "localhost:5000/my-busybox"},
	}

	for _, command := range commands {
		cmd := exec.Command("docker", command...)
		if b, err := cmd.CombinedOutput(); err != nil {
			t.Logf(string(b))
			t.Fatalf("command failed: %v", strings.Join(command, " "))
		}
	}
}

func KillRegistry(t *testing.T) {
	commands := [][]string{
		{"kill", "registry"},
		{"rm", "registry"},
	}

	for _, command := range commands {
		cmd := exec.Command("docker", command...)
		if b, err := cmd.CombinedOutput(); err != nil {
			t.Logf(string(b))
			t.Fatalf("command failed: %v", strings.Join(command, " "))
		}
	}
}
