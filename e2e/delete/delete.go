// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package delete

import (
	"bytes"
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/e2e/internal/testhelper"
)

type ctx struct {
	env e2e.TestEnv
}

func (c ctx) testDeleteCmd(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		agree      string
		expectExit int
	}{
		{
			name:       "delete unauthorized arch",
			args:       []string{"--arch=amd64", "library://test/default/test:v0.0.3"},
			agree:      "y",
			expectExit: 255,
		},
		{
			name:       "delete unauthorized no arch",
			args:       []string{"library://test/default/test:v0.0.3"},
			agree:      "y",
			expectExit: 255,
		},
		{
			name:       "delete disagree arch",
			args:       []string{"--arch=amd64", "library://test/default/test:v0.0.3"},
			agree:      "n",
			expectExit: 0,
		},
		{
			name:       "delete disagree noarch",
			args:       []string{"library://test/default/test:v0.0.3"},
			agree:      "n",
			expectExit: 0,
		},
		{
			name:       "delete unauthorized force arch",
			args:       []string{"--force", "--arch=amd64", "library://test/default/test:v0.0.3"},
			agree:      "",
			expectExit: 255,
		},
		{
			name:       "delete unauthorized force noarch",
			args:       []string{"--force", "library://test/default/test:v0.0.3"},
			agree:      "",
			expectExit: 255,
		},
		{
			name:       "delete unauthorized custom library",
			args:       []string{"--library=https://cloud.staging.sylabs.io", "library://test/default/test:v0.0.3"},
			agree:      "y",
			expectExit: 255,
		},
	}

	for _, tt := range tests {
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("delete"),
			e2e.WithArgs(tt.args...),
			e2e.WithStdin(bytes.NewBufferString(tt.agree)),
			e2e.ExpectExit(tt.expectExit),
		)
	}
}

// E2ETests is the main func to trigger the test suite.
func E2ETests(env e2e.TestEnv) testhelper.Tests {
	c := ctx{
		env: env,
	}

	return testhelper.Tests{
		"delete": c.testDeleteCmd,
	}
}
