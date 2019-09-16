// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package buildcfg

import (
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/e2e/internal/testhelper"
)

type buildcfgTests struct {
	env e2e.TestEnv
}

func (c buildcfgTests) buildcfgTests(t *testing.T) {
	tests := []struct {
		name           string
		cmdArgs        []string
		expectedOutput string
	}{
		{
			name:           "help",
			cmdArgs:        []string{"--help"},
			expectedOutput: "Output the currently set compile-time parameters",
		},
	}

	for _, tt := range tests {
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("buildcfg"),
			e2e.WithArgs(tt.cmdArgs...),
			e2e.ExpectExit(
				0,
				e2e.ExpectOutput(e2e.RegexMatch, `^`+tt.expectedOutput),
			),
		)
	}
}

// E2ETests is the main func to trigger the test suite
func E2ETests(env e2e.TestEnv) func(*testing.T) {
	c := buildcfgTests{
		env: env,
	}

	return testhelper.TestRunner(map[string]func(*testing.T){
		"buildcfgHelp": c.buildcfgTests,
	})
}
