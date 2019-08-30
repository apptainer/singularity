// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package delete

import (
	"bytes"
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
)

type ctx struct {
	env e2e.TestEnv
}

func (c *ctx) testDeleteCmd(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		agree      string
		expectExit int
	}{
		{
			name:       "delete unauthorized",
			args:       []string{"--arch=amd64", "library://test/default/test:v0.0.3"},
			agree:      "y",
			expectExit: 255,
		},
		{
			name:       "delete disagree",
			args:       []string{"--arch=amd64", "library://test/default/test:v0.0.3"},
			agree:      "n",
			expectExit: 0,
		},
		{
			name:       "delete without arch",
			args:       []string{"library://test/default/test:v0.0.3"},
			expectExit: 1,
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

// E2ETests is the main func to trigger the test suite
func E2ETests(env e2e.TestEnv) func(*testing.T) {
	c := &ctx{
		env: env,
	}

	return func(t *testing.T) {
		t.Run("delete", c.testDeleteCmd)
	}
}
