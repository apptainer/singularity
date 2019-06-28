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
	uuid "github.com/satori/go.uuid"
	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/internal/pkg/test/tool/require"
	"github.com/sylabs/singularity/pkg/ociruntime"
)

type ctx struct {
	env e2e.TestEnv
}

func checkOciState(t *testing.T, containerID string, state string) {
	checkStateFn := func(t *testing.T, r *e2e.SingularityCmdResult) {
		s := &ociruntime.State{}
		if err := json.Unmarshal(r.Stdout, s); err != nil {
			t.Errorf("can't unmarshal oci state output: %s", err)
			return
		}
		if s.Status != state {
			t.Errorf("bad container state returned, got %s instead of %s", s.Status, state)
		}
	}

	e2e.RunSingularity(
		t,
		e2e.WithPrivileges(true),
		e2e.WithCommand("oci state"),
		e2e.WithArgs(containerID),
		e2e.ExpectExit(0, checkStateFn),
	)
}

func genericOciMount(t *testing.T, c *ctx) (string, func()) {
	bundleDir, err := ioutil.TempDir(c.env.TestDir, "bundle-")
	if err != nil {
		t.Fatalf("failed to create bundle directory: %s", err)
	}
	ociConfig := filepath.Join(bundleDir, "config.json")

	e2e.RunSingularity(
		t,
		e2e.WithPrivileges(true),
		e2e.WithCommand("oci mount"),
		e2e.WithArgs(c.env.ImagePath, bundleDir),
		e2e.PostRun(func(t *testing.T) {
			if t.Failed() {
				t.Fatalf("failed to mount OCI bundle image")
			}

			g, err := generate.New(runtime.GOOS)
			if err != nil {
				t.Fatalf("failed to generate default OCI config: %s", err)
			}
			g.SetProcessTerminal(true)
			// NEED FIX: disable seccomp for circleci, ubuntu trusty
			// doesn't support syscalls _llseek and _newselect
			g.Config.Linux.Seccomp = nil

			err = g.SaveToFile(ociConfig, generate.ExportOptions{})
			if err != nil {
				t.Fatalf("failed to save OCI config: %s", err)
			}
		}),
		e2e.ExpectExit(0),
	)

	cleanup := func() {
		e2e.RunSingularity(
			t,
			e2e.WithPrivileges(true),
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
	e2e.RunSingularity(
		t,
		e2e.WithPrivileges(true),
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
	e2e.RunSingularity(
		t,
		e2e.WithPrivileges(true),
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

	e2e.RunSingularity(
		t,
		e2e.WithPrivileges(true),
		e2e.WithCommand("oci create"),
		e2e.WithArgs("-b", bundleDir, containerID),
		// this is required otherwise oci create hangs, this is
		// due to command Wait call that waits for the execution of a command
		// that closes standard file descriptors, but OCI create keeps
		// them to respect OCI runtime requirements
		e2e.ConsoleRun(),
		e2e.PostRun(func(t *testing.T) {
			if !t.Failed() {
				checkOciState(t, containerID, ociruntime.Created)
			}
		}),
		e2e.ExpectExit(0),
	)

	e2e.RunSingularity(
		t,
		"OciAttachStart",
		e2e.WithPrivileges(true),
		e2e.WithCommand("oci start"),
		e2e.WithArgs(containerID),
		e2e.PostRun(func(t *testing.T) {
			if !t.Failed() {
				checkOciState(t, containerID, ociruntime.Running)
			}
		}),
		e2e.ExpectExit(0),
	)

	e2e.RunSingularity(
		t,
		e2e.WithPrivileges(true),
		e2e.WithCommand("oci attach"),
		e2e.WithArgs(containerID),
		e2e.ConsoleRun(
			e2e.ConsoleSendLine("hostname"),
			e2e.ConsoleExpect("mrsdalloway"),
			e2e.ConsoleSendLine("exit"),
		),
		e2e.PostRun(func(t *testing.T) {
			if !t.Failed() {
				checkOciState(t, containerID, ociruntime.Stopped)
			}
		}),
		e2e.ExpectExit(0),
	)

	e2e.RunSingularity(
		t,
		e2e.WithPrivileges(true),
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

	e2e.RunSingularity(
		t,
		e2e.WithPrivileges(true),
		e2e.WithCommand("oci create"),
		e2e.WithArgs("-b", bundleDir, containerID),
		// this is required otherwise oci create hangs, this is
		// due to command Wait call that waits for the execution of a command
		// that closes standard file descriptors, but OCI create keeps
		// them to respect OCI runtime requirements
		e2e.ConsoleRun(),
		e2e.PostRun(func(t *testing.T) {
			if !t.Failed() {
				checkOciState(t, containerID, ociruntime.Created)
			}
		}),
		e2e.ExpectExit(0),
	)

	e2e.RunSingularity(
		t,
		"OciBasicStart",
		e2e.WithPrivileges(true),
		e2e.WithCommand("oci start"),
		e2e.WithArgs(containerID),
		e2e.PostRun(func(t *testing.T) {
			if !t.Failed() {
				checkOciState(t, containerID, ociruntime.Running)
			}
		}),
		e2e.ExpectExit(0),
	)

	e2e.RunSingularity(
		t,
		e2e.WithPrivileges(true),
		e2e.WithCommand("oci pause"),
		e2e.WithArgs(containerID),
		e2e.PreRun(func(t *testing.T) {
			// skip if cgroups freezer is not available
			require.CgroupsFreezer(t)
		}),
		e2e.PostRun(func(t *testing.T) {
			if !t.Failed() {
				checkOciState(t, containerID, ociruntime.Paused)
			}
		}),
		e2e.ExpectExit(0),
	)

	e2e.RunSingularity(
		t,
		e2e.WithPrivileges(true),
		e2e.WithCommand("oci resume"),
		e2e.WithArgs(containerID),
		e2e.PreRun(func(t *testing.T) {
			// skip if cgroups freezer is not available
			require.CgroupsFreezer(t)
		}),
		e2e.PostRun(func(t *testing.T) {
			if !t.Failed() {
				checkOciState(t, containerID, ociruntime.Running)
			}
		}),
		e2e.ExpectExit(0),
	)

	e2e.RunSingularity(
		t,
		e2e.WithPrivileges(true),
		e2e.WithCommand("oci start"),
		e2e.WithArgs(containerID),
		e2e.ExpectExit(255),
	)

	e2e.RunSingularity(
		t,
		e2e.WithPrivileges(true),
		e2e.WithCommand("oci exec"),
		e2e.WithArgs(containerID, "id"),
		e2e.ExpectExit(0),
	)

	e2e.RunSingularity(
		t,
		"OciBasicKill",
		e2e.WithPrivileges(true),
		e2e.WithCommand("oci kill"),
		e2e.WithArgs("-t", "2", containerID, "KILL"),
		e2e.PostRun(func(t *testing.T) {
			if !t.Failed() {
				checkOciState(t, containerID, ociruntime.Stopped)
			}
		}),
		e2e.ExpectExit(0),
	)

	e2e.RunSingularity(
		t,
		e2e.WithPrivileges(true),
		e2e.WithCommand("oci delete"),
		e2e.WithArgs(containerID),
		e2e.ExpectExit(0),
	)

	e2e.RunSingularity(
		t,
		e2e.WithPrivileges(true),
		e2e.WithCommand("oci state"),
		e2e.WithArgs(containerID),
		e2e.ExpectExit(255),
	)
	e2e.RunSingularity(
		t,
		e2e.WithPrivileges(true),
		e2e.WithCommand("oci kill"),
		e2e.WithArgs(containerID),
		e2e.ExpectExit(255),
	)
	e2e.RunSingularity(
		t,
		e2e.WithPrivileges(true),
		e2e.WithCommand("oci start"),
		e2e.WithArgs(containerID),
		e2e.ExpectExit(255),
	)
}

// RunE2ETests is the main func to trigger the test suite
func RunE2ETests(env e2e.TestEnv) func(*testing.T) {
	c := &ctx{
		env: env,
	}

	return func(t *testing.T) {
		t.Run("Basic", c.testOciBasic)
		t.Run("Attach", c.testOciAttach)
		t.Run("Run", c.testOciRun)
	}
}
