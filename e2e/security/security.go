// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package security

import (
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
)

type ctx struct {
	env e2e.TestEnv
}

// testSecurityUnpriv tests security flag fuctionality for singularity exec without elevated privileges
func (c *ctx) testSecurityUnpriv(t *testing.T) {
	tests := []struct {
		name       string
		image      string
		action     string
		argv       []string
		opts       []string
		expectExit int
	}{
		// taget UID/GID
		{
			name:       "Set_uid",
			image:      c.env.ImagePath,
			action:     "exec",
			argv:       []string{"id", "-u", "|", "grep", "99"},
			opts:       []string{"--security", "uid:99"},
			expectExit: 255,
		},
		{
			name:       "Set_gid",
			image:      c.env.ImagePath,
			action:     "exec",
			argv:       []string{"id", "-g", "|", "grep", "99"},
			opts:       []string{"gid:99"},
			expectExit: 255,
		},
		// seccomp from json file
		{
			name:       "SecComp_BlackList",
			image:      c.env.ImagePath,
			action:     "exec",
			argv:       []string{"mkdir", "/tmp/foo"},
			opts:       []string{"--security", "seccomp:./security/testdata/seccomp-profile.json"},
			expectExit: 159,
		},
		{
			name:       "SecComp_true",
			image:      c.env.ImagePath,
			action:     "exec",
			argv:       []string{"true"},
			opts:       []string{"--security", "seccomp:./security/testdata/seccomp-profile.json"},
			expectExit: 0,
		},
		// capabilities
		{
			name:       "capabilities_keep_true",
			image:      c.env.ImagePath,
			action:     "exec",
			argv:       []string{"ping", "-c", "1", "8.8.8.8"},
			opts:       []string{"--keep-privs"},
			expectExit: 255,
		},
		{
			name:       "capabilities_keep-false",
			image:      c.env.ImagePath,
			action:     "exec",
			argv:       []string{"ping", "-c", "1", "8.8.8.8"},
			expectExit: 1,
		},
		{
			name:       "capabilities_drop",
			image:      c.env.ImagePath,
			action:     "exec",
			argv:       []string{"ping", "-c", "1", "8.8.8.8"},
			opts:       []string{"--drop-caps", "CAP_NET_RAW"},
			expectExit: 1,
		},
	}

	verifyOutput := func(t *testing.T, r *e2e.SingularityCmdResult) {
		t.Logf("INFOOOOOOOOO: %s\n", r.Stdout)
	}

	for _, tt := range tests {
		optArgs := []string{}
		optArgs = append(optArgs, tt.opts...)
		optArgs = append(optArgs, c.env.ImagePath)
		optArgs = append(optArgs, tt.argv...)

		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithCommand("exec"),
			e2e.WithArgs(optArgs...),
			e2e.ExpectExit(tt.expectExit, verifyOutput),
			//e2e.ExpectExit(tt.expectExit, e2e.ExpectOutput(e2e.ContainMatch, tt.stdout)),
		)

	}
}

// RunE2ETests is the main func to trigger the test suite
func RunE2ETests(env e2e.TestEnv) func(*testing.T) {
	c := &ctx{
		env: env,
	}

	return func(t *testing.T) {
		//		e2e.EnsureImage(t, env)
		e2e.PullImage(t, env, "library://alpine", env.ImagePath)

		//		// We pull the two images required for the tests once
		//		e2e.PullImage(t, c.env, successURL, c.successImage)
		//		e2e.PullImage(t, c.env, corruptedURL, c.corruptedImage)

		t.Run("singularitysecurity", c.testSecurityUnpriv)
	}
}
