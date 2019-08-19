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
)

type ctx struct {
	env e2e.TestEnv
}

// testSecurityUnpriv tests the security flag fuctionality for singularity exec without elevated privileges
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
			expectExit: 2,
		},
		{
			name:       "capabilities_drop",
			image:      c.env.ImagePath,
			action:     "exec",
			argv:       []string{"ping", "-c", "1", "8.8.8.8"},
			opts:       []string{"--drop-caps", "CAP_NET_RAW"},
			expectExit: 2,
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

// testSecurityPriv tests security flag fuctionality for singularity exec with elevated privileges
func (c *ctx) testSecurityPriv(t *testing.T) {
	tests := []struct {
		name         string
		argv         []string
		opts         []string
		notExpecting string
		expectExit   int
	}{
		{
			name:       "TESTSSSSSSSSSSS",
			argv:       []string{"id"},
			opts:       []string{},
			expectExit: 0,
		},

		// taget UID/GID
		{
			name: "Set_uid",
			//argv: []string{"id", "-u"},
			argv: []string{"id", "-u", "|", "grep", "99"},
			opts: []string{"--security", "uid:99"},
			//notExpecting: "99",
			expectExit: 1,
		},
		{
			name: "Set_gid",
			argv: []string{"id", "-g", "|", "grep", "99"},
			//argv:         []string{"id", "-g"},
			opts: []string{"--security", "gid:99"},
			//notExpecting: "98",
			expectExit: 1,
		},
		// seccomp from json file
		{
			name:       "SecComp_BlackList",
			argv:       []string{"mkdir", "/tmp/foo"},
			opts:       []string{"--security", "seccomp:./testdata/seccomp-profile.json"},
			expectExit: 159,
		},
		{
			name:       "SecComp_true",
			argv:       []string{"true"},
			opts:       []string{"--security", "seccomp:./testdata/seccomp-profile.json"},
			expectExit: 0,
		},
		// capabilities
		{
			name:       "capabilities_keep",
			argv:       []string{"ping", "-c", "1", "8.8.8.8"},
			opts:       []string{"--keep-privs"},
			expectExit: 0,
		},
		{
			name:       "capabilities_drop",
			argv:       []string{"ping", "-c", "1", "8.8.8.8"},
			opts:       []string{"--drop-caps", "CAP_NET_RAW"},
			expectExit: 2,
		},
	}

	for _, tt := range tests {
		verifyOutput := func(t *testing.T, r *e2e.SingularityCmdResult) {
			t.Logf("INFOOOOOOOOO2222222: %s\n", string(r.Stdout))
			if tt.notExpecting != "" && tt.notExpecting != string(r.Stdout) {
				t.Fatalf("unexpected output; expecting: %s to not be: %s", tt.notExpecting, string(r.Stdout))
			}
		}

		optArgs := []string{}
		optArgs = append(optArgs, tt.opts...)
		optArgs = append(optArgs, c.env.ImagePath)
		optArgs = append(optArgs, tt.argv...)

		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithPrivileges(true),
			e2e.WithCommand("exec"),
			e2e.WithArgs(optArgs...),
			e2e.ExpectExit(tt.expectExit, verifyOutput),
		)

	}
}

// testSecurityConfOwnership tests checks on config files ownerships
func (c *ctx) testSecurityConfOwnership(t *testing.T) {
	configFile := buildcfg.SINGULARITY_CONF_FILE

	//	// Change file ownership (do not try this at home)
	//	err := os.Chown(configFile, 1001, 0)
	//	if err != nil {
	//		t.Fatalf("failed to change owner for: %s: %s", configFile, err)
	//	}

	// try to run
	c.env.RunSingularity(
		t,
		e2e.AsSubtest("non root config"),
		e2e.WithPrivileges(true),
		e2e.PreRun(func(t *testing.T) {
			e2e.Privileged(func(t *testing.T) {
				// Change file ownership (do not try this at home)
				err := os.Chown(configFile, 1001, 0)
				if err != nil {
					t.Fatalf("failed to change owner for: %s: %s", configFile, err)
				}
			})
		}),
		e2e.PostRun(func(t *testing.T) {
			e2e.Privileged(func(t *testing.T) {
				// return file ownership to normal
				err := os.Chown(configFile, 0, 0)
				if err != nil {
					t.Fatalf("failed to change config file owner to root: %s: %s", configFile, err)
				}
			})
		}),
		e2e.WithCommand("exec"),
		e2e.WithArgs([]string{c.env.ImagePath, "/bin/true"}...),
		e2e.ExpectExit(0),
	)

	//	// return file ownership to normal
	//	err := os.Chown(configFile, 0, 0)
	//	if err != nil {
	//		t.Fatalf("failed to change config file owner to root: %s: %s", configFile, err)
	//	}
}

// RunE2ETests is the main func to trigger the test suite
func RunE2ETests(env e2e.TestEnv) func(*testing.T) {
	c := &ctx{
		env: env,
	}

	return func(t *testing.T) {
		e2e.PullImage(t, env, "library://westleyk/tests/ubuntu_ping:v1.0", env.ImagePath)

		t.Run("singularitySecurityUnpriv", c.testSecurityUnpriv)
		t.Run("singularitySecurityPriv", c.testSecurityPriv)
		//		t.Run("testSecurityConfOwnership", c.testSecurityConfOwnership)
	}
}
