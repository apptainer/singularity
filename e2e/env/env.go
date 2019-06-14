// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// This test sets singularity image specific environment variables and
// verifies that they are properly set.

package singularityenv

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/internal/pkg/test"
	"github.com/sylabs/singularity/internal/pkg/test/exec"
)

type testingEnv struct {
	// base env for running tests
	CmdPath string `split_words:"true"`
}

var testenv testingEnv

func singularityEnv(t *testing.T) {
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

			cmd := exec.Command(testenv.CmdPath, args..., append(os.Environ(), currentTest.env...))

			out := cmd.Run(t).Stdout()
			t.Logf("args: '%v'", strings.Join(args, " "))
			t.Logf("env: '%v'", strings.Join(cmd.Env, " "))
			t.Log(out)

			if err != nil {
				t.Fatalf("Error running command: %v", err)
			}

			fmt.Println("Current test path is " + currentTest.path)
			if !strings.Contains(out, currentTest.path) {
				t.Fatalf("Command output did not contain the path '%s'", currentTest.path)
			}
		}))
	}
}

// RunE2ETests is the main func to trigger the test suite
func RunE2ETests(t *testing.T) {
	e2e.LoadEnv(t, &testenv)

	t.Log(testenv)

	// try to build from a non existen path
	t.Run("singularityEnv", singularityEnv)
}
