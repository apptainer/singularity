// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package oci

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/opencontainers/runtime-tools/generate"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/internal/pkg/test/tool/require"
	"github.com/sylabs/singularity/pkg/ociruntime"
)

type ctx struct {
	env e2e.TestEnv
}

func (c *ctx) checkOciState(t *testing.T, containerID, state string) {
	checkStateFn := func(t *testing.T, r *e2e.SingularityCmdResult) {
		s := &ociruntime.State{}
		if err := json.Unmarshal(r.Stdout, s); err != nil {
			err = errors.Wrapf(err, "unmarshaling OCI state from JSON: %s", r.Stdout)
			t.Errorf("can't unmarshal oci state output: %+v", err)
			return
		}
		if s.Status != state {
			t.Errorf("bad container state returned, got %s instead of %s", s.Status, state)
		}
	}

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.RootProfile),
		e2e.WithCommand("oci state"),
		e2e.WithArgs(containerID),
		e2e.ExpectExit(0, checkStateFn),
	)
}

func genericOciMount(t *testing.T, c *ctx) (string, func()) {
	bundleDir, err := ioutil.TempDir(c.env.TestDir, "bundle-")
	if err != nil {
		err = errors.Wrapf(err, "creating temporary bundle directory at %q", c.env.TestDir)
		t.Fatalf("failed to create bundle directory: %+v", err)
	}
	ociConfig := filepath.Join(bundleDir, "config.json")

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.RootProfile),
		e2e.WithCommand("oci mount"),
		e2e.WithArgs(c.env.ImagePath, bundleDir),
		e2e.PostRun(func(t *testing.T) {
			if t.Failed() {
				t.Fatalf("failed to mount OCI bundle image")
			}

			g, err := generate.New(runtime.GOOS)
			if err != nil {
				err = errors.Wrapf(err, "generating default OCI config for %q", runtime.GOOS)
				t.Fatalf("failed to generate default OCI config: %+v", err)
			}
			g.SetProcessTerminal(true)
			// NEED FIX: disable seccomp for circleci, ubuntu trusty
			// doesn't support syscalls _llseek and _newselect
			// g.Config.Linux.Seccomp = nil

			err = g.SaveToFile(ociConfig, generate.ExportOptions{})
			if err != nil {
				err = errors.Wrapf(err, "saving OCI config at %q", ociConfig)
				t.Fatalf("failed to save OCI config: %+v", err)
			}
		}),
		e2e.ExpectExit(0),
	)

	cleanup := func() {
		c.env.RunSingularity(
			t,
			e2e.WithProfile(e2e.RootProfile),
			e2e.WithCommand("oci umount"),
			e2e.WithArgs(bundleDir),
			e2e.ExpectExit(0),
		)
	}

	return bundleDir, cleanup
}

func (c *ctx) testOciRun(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	containerID := uuid.NewV4().String()
	bundleDir, umountFn := genericOciMount(t, c)

	// umount bundle
	defer umountFn()

	// oci run -b
	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.RootProfile),
		e2e.WithCommand("oci run"),
		e2e.WithArgs("-b", bundleDir, containerID),
		e2e.ConsoleRun(
			e2e.ConsoleSendLine("hostname"),
			e2e.ConsoleExpect("mrsdalloway"),
			e2e.ConsoleSendLine("id -un"),
			e2e.ConsoleExpect("root"),
			e2e.ConsoleSendLine("exit"),
		),
		e2e.ExpectExit(0),
	)

	// oci state should fail
	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.RootProfile),
		e2e.WithCommand("oci state"),
		e2e.WithArgs(containerID),
		e2e.ExpectExit(255),
	)
}

func (c *ctx) testOciAttach(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	containerID := uuid.NewV4().String()
	bundleDir, umountFn := genericOciMount(t, c)

	// umount bundle
	defer umountFn()

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.RootProfile),
		e2e.WithCommand("oci create"),
		e2e.WithArgs("-b", bundleDir, containerID),
		// this is required otherwise oci create hangs, this is
		// due to command Wait call that waits for the execution of a command
		// that closes standard file descriptors, but OCI create keeps
		// them to respect OCI runtime requirements
		e2e.ConsoleRun(),
		e2e.PostRun(func(t *testing.T) {
			if !t.Failed() {
				c.checkOciState(t, containerID, ociruntime.Created)
			}
		}),
		e2e.ExpectExit(0),
	)

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.RootProfile),
		e2e.WithCommand("oci start"),
		e2e.WithArgs(containerID),
		e2e.PostRun(func(t *testing.T) {
			if !t.Failed() {
				c.checkOciState(t, containerID, ociruntime.Running)
			}
		}),
		e2e.ExpectExit(0),
	)

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.RootProfile),
		e2e.WithCommand("oci attach"),
		e2e.WithArgs(containerID),
		e2e.ConsoleRun(
			e2e.ConsoleSendLine("hostname"),
			e2e.ConsoleExpect("mrsdalloway"),
			e2e.ConsoleSendLine("exit"),
		),
		e2e.PostRun(func(t *testing.T) {
			if !t.Failed() {
				c.checkOciState(t, containerID, ociruntime.Stopped)
			}
		}),
		e2e.ExpectExit(0),
	)

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.RootProfile),
		e2e.WithCommand("oci delete"),
		e2e.WithArgs(containerID),
		e2e.ExpectExit(0),
	)
}

func (c *ctx) testOciBasic(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	containerID := uuid.NewV4().String()
	bundleDir, umountFn := genericOciMount(t, c)

	// umount bundle
	defer umountFn()

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.RootProfile),
		e2e.WithCommand("oci create"),
		e2e.WithArgs("-b", bundleDir, containerID),
		// this is required otherwise oci create hangs, this is
		// due to command Wait call that waits for the execution of a command
		// that closes standard file descriptors, but OCI create keeps
		// them to respect OCI runtime requirements
		e2e.ConsoleRun(),
		e2e.PostRun(func(t *testing.T) {
			if !t.Failed() {
				c.checkOciState(t, containerID, ociruntime.Created)
			}
		}),
		e2e.ExpectExit(0),
	)

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.RootProfile),
		e2e.WithCommand("oci start"),
		e2e.WithArgs(containerID),
		e2e.PostRun(func(t *testing.T) {
			if !t.Failed() {
				c.checkOciState(t, containerID, ociruntime.Running)
			}
		}),
		e2e.ExpectExit(0),
	)

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.RootProfile),
		e2e.WithCommand("oci pause"),
		e2e.WithArgs(containerID),
		e2e.PreRun(func(t *testing.T) {
			// skip if cgroups freezer is not available
			require.CgroupsFreezer(t)
		}),
		e2e.PostRun(func(t *testing.T) {
			if !t.Failed() {
				c.checkOciState(t, containerID, ociruntime.Paused)
			}
		}),
		e2e.ExpectExit(0),
	)

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.RootProfile),
		e2e.WithCommand("oci resume"),
		e2e.WithArgs(containerID),
		e2e.PreRun(func(t *testing.T) {
			// skip if cgroups freezer is not available
			require.CgroupsFreezer(t)
		}),
		e2e.PostRun(func(t *testing.T) {
			if !t.Failed() {
				c.checkOciState(t, containerID, ociruntime.Running)
			}
		}),
		e2e.ExpectExit(0),
	)

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.RootProfile),
		e2e.WithCommand("oci start"),
		e2e.WithArgs(containerID),
		e2e.ExpectExit(255),
	)

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.RootProfile),
		e2e.WithCommand("oci exec"),
		e2e.WithArgs(containerID, "id"),
		e2e.ExpectExit(0),
	)

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.RootProfile),
		e2e.WithCommand("oci kill"),
		e2e.WithArgs("-t", "2", containerID, "KILL"),
		e2e.PostRun(func(t *testing.T) {
			if !t.Failed() {
				c.checkOciState(t, containerID, ociruntime.Stopped)
			}
		}),
		e2e.ExpectExit(0),
	)

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.RootProfile),
		e2e.WithCommand("oci delete"),
		e2e.WithArgs(containerID),
		e2e.ExpectExit(0),
	)

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.RootProfile),
		e2e.WithCommand("oci state"),
		e2e.WithArgs(containerID),
		e2e.ExpectExit(255),
	)
	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.RootProfile),
		e2e.WithCommand("oci kill"),
		e2e.WithArgs(containerID),
		e2e.ExpectExit(255),
	)
	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.RootProfile),
		e2e.WithCommand("oci start"),
		e2e.WithArgs(containerID),
		e2e.ExpectExit(255),
	)
}

func (c *ctx) testOciHelp(t *testing.T) {
	tests := []struct {
		name          string
		expectedRegex string
	}{
		{
			name:          "attach",
			expectedRegex: `^Attach console to a running container process \(root user only\)`,
		},
		{
			name:          "create",
			expectedRegex: `^Create a container from a bundle directory \(root user only\)`,
		},
		{
			name:          "delete",
			expectedRegex: `^Delete container \(root user only\)`,
		},
		{
			name:          "exec",
			expectedRegex: `^Execute a command within container \(root user only\)`,
		},
		{
			name:          "kill",
			expectedRegex: `^Kill a container \(root user only\)`,
		},
		{
			name:          "mount",
			expectedRegex: `^Mount create an OCI bundle from SIF image \(root user only\)`,
		},
		{
			name:          "pause",
			expectedRegex: `^Suspends all processes inside the container \(root user only\)`,
		},
		{
			name:          "resume",
			expectedRegex: `^Resumes all processes previously paused inside the container \(root user only\)`,
		},
		{
			name:          "run",
			expectedRegex: `^Create/start/attach/delete a container from a bundle directory \(root user only\)`,
		},
		{
			name:          "start",
			expectedRegex: `^Start container process \(root user only\)`,
		},
		{
			name:          "umount",
			expectedRegex: `^Umount delete bundle \(root user only\)`,
		},
		{
			name:          "update",
			expectedRegex: `^Update container cgroups resources \(root user only\)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c.env.RunSingularity(
				t,
				e2e.AsSubtest(tt.name),
				e2e.WithProfile(e2e.UserProfile),
				e2e.WithCommand("oci"),
				e2e.WithArgs([]string{tt.name, "--help"}...),
				e2e.ExpectExit(
					0,
					e2e.ExpectOutput(e2e.RegexMatch, tt.expectedRegex),
				),
			)
		})
	}
}

// E2ETests is the main func to trigger the test suite
func E2ETests(env e2e.TestEnv) func(*testing.T) {
	c := &ctx{
		env: env,
	}

	return func(t *testing.T) {
		t.Run("Basic", c.testOciBasic)
		t.Run("Attach", c.testOciAttach)
		t.Run("Run", c.testOciRun)
		t.Run("Help", c.testOciHelp)
	}
}
