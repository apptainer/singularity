// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package security

import (
	"os"
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/test/tool/require"
)

type ctx struct {
	env e2e.TestEnv
}

// testSecurityUnpriv tests the security flag fuctionality for singularity exec without elevated privileges
func (c ctx) testSecurityUnpriv(t *testing.T) {
	tests := []struct {
		name       string
		image      string
		argv       []string
		opts       []string
		preFn      func(*testing.T)
		expectOp   e2e.SingularityCmdResultOp
		expectExit int
	}{
		// taget UID/GID
		{
			name:       "Set_uid",
			argv:       []string{"id", "-u"},
			opts:       []string{"--security", "uid:99"},
			expectExit: 255,
			// TODO: add expect stderr for "uid security feature requires root privileges"
			// pending issue: https://github.com/sylabs/singularity/issues/4280
		},
		{
			name:       "Set_gid",
			argv:       []string{"id", "-g"},
			opts:       []string{"--security", "gid:99"},
			expectExit: 255,
		},
		// seccomp from json file
		{
			name:       "SecComp_BlackList",
			argv:       []string{"mkdir", "/tmp/foo"},
			opts:       []string{"--security", "seccomp:./security/testdata/seccomp-profile.json"},
			preFn:      require.Seccomp,
			expectExit: 159, // process should be killed with SIGSYS (128+31)
		},
		{
			name:       "SecComp_true",
			argv:       []string{"true"},
			opts:       []string{"--security", "seccomp:./security/testdata/seccomp-profile.json"},
			preFn:      require.Seccomp,
			expectExit: 0,
		},
		// capabilities
		{
			name:       "capabilities_keep_true",
			argv:       []string{"grep", "^CapEff:", "/proc/self/status"},
			opts:       []string{"--keep-privs"},
			expectExit: 255, // singularity errors out, --keep-privs needs root
		},
		{
			// we start without any capabilities, the
			// expected set is empty.
			name:     "capabilities_keep-false",
			argv:     []string{"grep", "^CapEff:", "/proc/self/status"},
			expectOp: e2e.ExpectOutput(e2e.RegexMatch, `CapEff:\s+0+\n`),
		},
		{
			// this one is tricky: we are asking to drop
			// cap_net_raw, but since we start without any
			// capabilities the expected set is empty.
			name:     "capabilities_drop",
			argv:     []string{"grep", "^Cap.*:", "/proc/self/status"},
			opts:     []string{"--drop-caps", "CAP_NET_RAW"},
			expectOp: e2e.ExpectOutput(e2e.RegexMatch, `CapEff:\s+0+\n`),
		},
	}

	e2e.EnsureImage(t, c.env)

	for _, tt := range tests {
		optArgs := []string{}
		optArgs = append(optArgs, tt.opts...)
		optArgs = append(optArgs, c.env.ImagePath)
		optArgs = append(optArgs, tt.argv...)

		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("exec"),
			e2e.WithArgs(optArgs...),
			e2e.PreRun(tt.preFn),
			e2e.ExpectExit(tt.expectExit, tt.expectOp),
		)

	}
}

// testSecurityPriv tests security flag fuctionality for singularity exec with elevated privileges
func (c ctx) testSecurityPriv(t *testing.T) {
	tests := []struct {
		name       string
		argv       []string
		opts       []string
		preFn      func(*testing.T)
		expectOp   e2e.SingularityCmdResultOp
		expectExit int
	}{
		// taget UID/GID
		{
			name:       "Set_uid",
			argv:       []string{"id", "-u"},
			opts:       []string{"--security", "uid:99"},
			expectOp:   e2e.ExpectOutput(e2e.ExactMatch, "99"),
			expectExit: 0,
		},
		{
			name:       "Set_gid",
			argv:       []string{"id", "-g"},
			opts:       []string{"--security", "gid:99"},
			expectOp:   e2e.ExpectOutput(e2e.ExactMatch, "99"),
			expectExit: 0,
		},
		// seccomp from json file
		{
			name:       "SecComp_BlackList",
			argv:       []string{"mkdir", "/tmp/foo"},
			opts:       []string{"--security", "seccomp:./testdata/seccomp-profile.json"},
			preFn:      require.Seccomp,
			expectExit: 159, // process should be killed with SIGSYS (128+31)
		},
		{
			name:       "SecComp_true",
			argv:       []string{"true"},
			opts:       []string{"--security", "seccomp:./testdata/seccomp-profile.json"},
			preFn:      require.Seccomp,
			expectExit: 0,
		},
		// capabilities
		{
			// this might break if new capabilities are
			// added
			//
			// TODO(mem): what we really want to test is
			// that the set outside the container is the
			// same set inside the container.
			name:     "capabilities_keep",
			argv:     []string{"grep", "^CapEff:", "/proc/self/status"},
			opts:     []string{"--keep-privs"},
			expectOp: e2e.ExpectOutput(e2e.RegexMatch, `CapEff:\s+0000003fffffffff\n`),
		},
		{
			// this might break if new capabilities are
			// added; cap_net_raw corresponds to the missing
			// bit in CapEff.
			//
			// TODO(mem): what we really want to test is
			// that the set outside the container is the
			// same set inside the container, modulo
			// cap_net_raw.
			name:     "capabilities_drop",
			argv:     []string{"grep", "^CapEff:", "/proc/self/status"},
			opts:     []string{"--drop-caps", "CAP_NET_RAW"},
			expectOp: e2e.ExpectOutput(e2e.RegexMatch, `CapEff:\s+0000003fffffdfff\n`),
		},
	}

	e2e.EnsureImage(t, c.env)

	for _, tt := range tests {
		optArgs := []string{}
		optArgs = append(optArgs, tt.opts...)
		optArgs = append(optArgs, c.env.ImagePath)
		optArgs = append(optArgs, tt.argv...)

		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.RootProfile),
			e2e.WithCommand("exec"),
			e2e.WithArgs(optArgs...),
			e2e.PreRun(tt.preFn),
			e2e.ExpectExit(tt.expectExit, tt.expectOp),
		)

	}
}

// testSecurityConfOwnership tests checks on config files ownerships
func (c ctx) testSecurityConfOwnership(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	configFile := buildcfg.SINGULARITY_CONF_FILE

	c.env.RunSingularity(
		t,
		e2e.AsSubtest("non root config"),
		e2e.WithProfile(e2e.UserProfile),
		e2e.PreRun(func(t *testing.T) {
			e2e.Privileged(func(t *testing.T) {
				// Change file ownership (do not try this at home)
				err := os.Chown(configFile, 1001, 0)
				if err != nil {
					t.Fatalf("failed to change owner for: %s: %s", configFile, err)
				}
			})(t)
		}),
		e2e.PostRun(func(t *testing.T) {
			e2e.Privileged(func(t *testing.T) {
				// return file ownership to normal
				err := os.Chown(configFile, 0, 0)
				if err != nil {
					t.Fatalf("failed to change config file owner to root: %s: %s", configFile, err)
				}
			})(t)
		}),
		e2e.WithCommand("exec"),
		e2e.WithArgs([]string{c.env.ImagePath, "/bin/true"}...),
		e2e.ExpectExit(255),
	)
}

// E2ETests is the main func to trigger the test suite
func E2ETests(env e2e.TestEnv) func(*testing.T) {
	c := ctx{
		env: env,
	}

	return func(t *testing.T) {
		t.Run("singularitySecurityUnpriv", c.testSecurityUnpriv)
		t.Run("singularitySecurityPriv", c.testSecurityPriv)
		t.Run("testSecurityConfOwnership", c.testSecurityConfOwnership)
	}
}
