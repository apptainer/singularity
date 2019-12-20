// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package runhelp

import (
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/e2e/internal/testhelper"
)

type ctx struct {
	env e2e.TestEnv
}

func (c ctx) testRunHelp(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	tests := []struct {
		name   string
		argv   []string
		output string
		exit   int
	}{
		{
			name:   "DefaultHelp",
			argv:   []string{c.env.ImagePath},
			output: "BAD_GUY=Thanos",
			exit:   0,
		},
		{
			name:   "AppFooHelp",
			argv:   []string{"--app", "foo", c.env.ImagePath},
			output: "This is the help for foo!",
			exit:   0,
		},
		{
			name:   "AppFakeHelp",
			argv:   []string{"--app", "fake", c.env.ImagePath},
			output: "No help sections were defined for this image",
			exit:   0,
		},
		{
			name: "NoImage",
			argv: []string{"/fake/image"},
			exit: 255,
		},
	}

	for _, tt := range tests {
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("run-help"),
			e2e.WithArgs(tt.argv...),
			e2e.ExpectExit(
				tt.exit,
				e2e.ExpectOutput(e2e.ContainMatch, tt.output),
			),
		)
	}
}

// E2ETests is the main func to trigger the test suite
func E2ETests(env e2e.TestEnv) testhelper.Tests {
	c := ctx{
		env: env,
	}

	return testhelper.Tests{
		"run-help command": c.testRunHelp,
	}
}
