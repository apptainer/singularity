// Copyright (c) 2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularityenv

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/hpcng/singularity/e2e/internal/e2e"
	"github.com/hpcng/singularity/internal/pkg/buildcfg"
	"github.com/hpcng/singularity/internal/pkg/util/fs"
	"github.com/hpcng/singularity/pkg/util/rlimit"
)

// Check that an old-style `/environment` file is interpreted
// and can set PATH.
func (c ctx) issue5426(t *testing.T) {
	sandboxDir, cleanup := e2e.MakeTempDir(t, c.env.TestDir, "sandbox-", "")
	defer cleanup(t)

	// Build a current sandbox
	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("build"),
		e2e.WithArgs("--force", "--sandbox", sandboxDir, "library://alpine:3.11.5"),
		e2e.ExpectExit(0),
	)

	// Remove the /.singularity.d
	if err := os.RemoveAll(path.Join(sandboxDir, ".singularity.d")); err != nil {
		t.Fatalf("Could not remove sandbox /.singularity.d: %s", err)
	}
	// Remove the /environment symlink
	if err := os.Remove(path.Join(sandboxDir, "environment")); err != nil {
		t.Fatalf("Could not remove sandbox /environment symlink: %s", err)
	}
	// Copy in the test environment file
	testEnvironment := path.Join("testdata", "regressions", "legacy-environment")
	if err := fs.CopyFile(testEnvironment, path.Join(sandboxDir, "environment"), 0755); err != nil {
		t.Fatalf("Could not add legacy /environment to sandbox: %s", err)
	}

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("exec"),
		e2e.WithArgs(sandboxDir, "/bin/sh", "-c", "echo $PATH"),
		e2e.ExpectExit(
			0,
			e2e.ExpectOutput(e2e.ContainMatch, "/canary/path")),
	)
}

// Check that we hit engine configuation size limit with a rather big
// configuration by passing some big environment variables.
func (c ctx) issue5057(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	cur, _, err := rlimit.Get("RLIMIT_STACK")
	if err != nil {
		t.Fatalf("Could not determine stack size limit: %s", err)
	}
	if buildcfg.MAX_ENGINE_CONFIG_SIZE >= cur/4 {
		t.Skipf("stack limit too low")
	}

	max := uint64(buildcfg.MAX_CHUNK_SIZE)

	big := make([]byte, max)
	for i := uint64(0); i < max; i++ {
		big[i] = 'A'
	}
	bigEnv := make([]string, buildcfg.MAX_ENGINE_CONFIG_CHUNK)
	for i := range bigEnv {
		bigEnv[i] = fmt.Sprintf("B%d=%s", i, string(big))
	}

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("exec"),
		e2e.WithEnv(bigEnv),
		e2e.WithArgs(c.env.ImagePath, "true"),
		e2e.ExpectExit(255),
	)

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("exec"),
		e2e.WithEnv(bigEnv[:buildcfg.MAX_ENGINE_CONFIG_CHUNK-1]),
		e2e.WithArgs(c.env.ImagePath, "true"),
		e2e.ExpectExit(0),
	)
}

// If a $ in a SINGULARITYENV_ env var is escaped, it should become a
// literal $ in the container env var.
// This allows setting e.g. LD_PRELOAD=/foo/bar/$LIB/baz.so
func (c ctx) issue43(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	env := []string{`SINGULARITYENV_LD_PRELOAD=/foo/bar/\$LIB/baz.so`}
	args := []string{c.env.ImagePath, "/bin/sh", "-c", "echo \"${LD_PRELOAD}\""}

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("exec"),
		e2e.WithEnv(env),
		e2e.WithArgs(args...),
		e2e.ExpectExit(
			0,
			e2e.ExpectOutput(e2e.ExactMatch, `/foo/bar/$LIB/baz.so`),
		),
	)
}

// https://github.com/sylabs/singularity/issues/274
// The conda profile.d script must be able to be source'd from %environment.
// This has been broken by changes to mvdan.cc/sh interacting badly with our
// custom internalExecHandler.
// The test is quite heavyweight, but is warranted IMHO to ensure that conda
// environment activation works as expected, as this is a common use-case
// for SingularityCE.
func (c ctx) issue274(t *testing.T) {
	imageDir, cleanup := e2e.MakeTempDir(t, c.env.TestDir, "issue274-", "")
	defer cleanup(t)
	imagePath := filepath.Join(imageDir, "container")

	// Create a minimal conda environment on the current miniconda3 base.
	// Source the conda profile.d code and activate the env from `%environment`.
	def := `Bootstrap: docker
From: continuumio/miniconda3:latest

%post

	. /opt/conda/etc/profile.d/conda.sh
	conda create -n env python=3

%environment

	source /opt/conda/etc/profile.d/conda.sh
	conda activate env
`
	defFile, err := e2e.WriteTempFile(imageDir, "deffile", def)
	if err != nil {
		t.Fatalf("Unable to create test definition file: %v", err)
	}

	c.env.RunSingularity(
		t,
		e2e.AsSubtest("build"),
		e2e.WithProfile(e2e.RootProfile),
		e2e.WithCommand("build"),
		e2e.WithArgs(imagePath, defFile),
		e2e.ExpectExit(0),
	)
	// An exec of `conda info` in the container should show environment active, no errors.
	// I.E. the `%environment` section should have worked.
	c.env.RunSingularity(
		t,
		e2e.AsSubtest("exec"),
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("exec"),
		e2e.WithArgs(imagePath, "conda", "info"),
		e2e.ExpectExit(0,
			e2e.ExpectOutput(e2e.ContainMatch, "active environment : env"),
			e2e.ExpectError(e2e.ExactMatch, ""),
		),
	)
}
