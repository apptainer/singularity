// Copyright (c) 2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularityenv

import (
	"os"
	"path"
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
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
