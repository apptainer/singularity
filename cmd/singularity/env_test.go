// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// This file has been deprecated and will disappear with version 3.3
// of singularity. The functionality has been moved to e2e/env/env.go

package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
)

func TestSingularityEnv(t *testing.T) {
	// Singularity defines a path by default. See singularityware/singularity/etc/init.
	var defaultImage = "docker://alpine:3.8"
	var defaultPath = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

	// This image sets a custom path.
	var customImage = "docker://godlovedc/lolcow"
	var customPath = "/usr/games:" + defaultPath

	// Append or prepend this path.
	var partialPath = "/foo"

	// Overwrite the path with this one.
	var overwrittenPath = "/usr/bin:/bin"

	var singularityEnvTests = []struct {
		name  string
		image string
		path  string
		env   []string
	}{
		{"DefaultPath", defaultImage, defaultPath, []string{}},
		{"CustomPath", customImage, customPath, []string{}},
		{"AppendToDefaultPath", defaultImage, defaultPath + ":" + partialPath, []string{"SINGULARITYENV_APPEND_PATH=/foo"}},
		{"AppendToCustomPath", customImage, customPath + ":" + partialPath, []string{"SINGULARITYENV_APPEND_PATH=/foo"}},
		{"PrependToDefaultPath", defaultImage, partialPath + ":" + defaultPath, []string{"SINGULARITYENV_PREPEND_PATH=/foo"}},
		{"PrependToCustomPath", customImage, partialPath + ":" + customPath, []string{"SINGULARITYENV_PREPEND_PATH=/foo"}},
		{"OverwriteDefaultPath", defaultImage, overwrittenPath, []string{"SINGULARITYENV_PATH=" + overwrittenPath}},
		{"OverwriteCustomPath", customImage, overwrittenPath, []string{"SINGULARITYENV_PATH=" + overwrittenPath}},
	}

	for _, currentTest := range singularityEnvTests {
		t.Run(currentTest.name, test.WithoutPrivilege(func(t *testing.T) {
			args := []string{"exec", currentTest.image, "env"}

			cmd := exec.Command(cmdPath, args...)
			cmd.Env = append(os.Environ(), currentTest.env...)
			b, err := cmd.CombinedOutput()

			out := string(b)
			t.Logf("args: '%v'", strings.Join(args, " "))
			t.Logf("env: '%v'", strings.Join(cmd.Env, " "))
			t.Log(out)

			if err != nil {
				t.Fatalf("Error running command: %v", err)
			}

			if !strings.Contains(out, currentTest.path) {
				t.Fatalf("Command output did not contain the path '%s'", currentTest.path)
			}
		}))
	}
}
