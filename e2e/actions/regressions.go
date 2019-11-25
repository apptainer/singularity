// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package actions

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
)

// Check there is no file descriptor leaked in the container
// process. This test expect 4 file descriptors, 3 for stdin,
// stdout, stderr and one opened by the ls command.
func (c actionTests) issue4488(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("exec"),
		e2e.WithArgs(c.env.ImagePath, "ls", "-1", "/proc/self/fd"),
		e2e.ExpectExit(
			0,
			e2e.ExpectOutput(e2e.ExactMatch, "0\n1\n2\n3"),
		),
	)
}

// Check that current working directory when is the user
// home directory doesn't override the custom home directory.
func (c actionTests) issue4587(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	u := e2e.UserProfile.HostUser(t)

	homeDir, cleanup := e2e.MakeTempDir(t, c.env.TestDir, "homedir-", "")
	defer cleanup(t)

	canaryFile := filepath.Join(homeDir, "canary_file")
	if err := fs.Touch(canaryFile); err != nil {
		t.Fatalf("failed to create canary file: %s", err)
	}

	homeBind := homeDir + ":" + u.Dir

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithDir(u.Dir),
		e2e.WithCommand("exec"),
		e2e.WithArgs("--home", homeBind, c.env.ImagePath, "test", "-f", filepath.Join(u.Dir, "canary_file")),
		e2e.ExpectExit(0),
	)
}

// Check that current working directory doesn't interfere
// with image content when using underlay.
func (c actionTests) issue4755(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	sandbox, cleanup := e2e.MakeTempDir(t, c.env.TestDir, "sandbox-", "")
	defer cleanup(t)

	// convert test image to sandbox
	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("build"),
		e2e.WithArgs("--force", "--sandbox", sandbox, c.env.ImagePath),
		e2e.ExpectExit(0),
	)

	// create a file in image /tmp in order to trigger the issue
	// with underlay layer
	baseDir := filepath.Join(sandbox, filepath.Dir(c.env.TestDir))
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		t.Fatalf("can't create image directory %s: %s", baseDir, err)
	}
	path := filepath.Join(baseDir, "underlay-test")
	if err := fs.Touch(path); err != nil {
		t.Fatalf("can't create %s: %s", path, err)
	}

	// use of user namespace to force runtime to use underlay
	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserNamespaceProfile),
		e2e.WithDir(c.env.TestDir),
		e2e.WithCommand("exec"),
		e2e.WithArgs(sandbox, "true"),
		e2e.ExpectExit(0),
	)
}

// Check that the last element of current working directory when it's
// a symlink pointing to a relative target is correctly handled by the
// runtime.
func (c actionTests) issue4768(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	homeDir, cleanup := e2e.MakeTempDir(t, c.env.TestDir, "issue-4768-", "")
	defer cleanup(t)

	symCwdPath := filepath.Join(homeDir, "symlink")
	if err := os.Symlink(".", symCwdPath); err != nil {
		t.Fatalf("failed to create symlink %s: %s", symCwdPath, err)
	}

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithDir(symCwdPath),
		e2e.WithCommand("exec"),
		e2e.WithArgs(c.env.ImagePath, "pwd"),
		e2e.ExpectExit(
			0,
			e2e.ExpectOutput(e2e.ExactMatch, homeDir),
		),
	)
}
