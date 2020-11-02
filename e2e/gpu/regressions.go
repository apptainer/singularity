// Copyright (c) 2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package gpu

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
)

// Check that we fall back to system ldconfig if a non-working one is on PATH
func (c ctx) issue5002(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	// Create a dummy ldconfig that doesn't work
	tmpDir, cleanup := e2e.MakeTempDir(t, c.env.TestDir, "issue-5631-", "")
	defer e2e.Privileged(cleanup)(t)
	fakeLdconfig := filepath.Join(tmpDir, "ldconfig")
	err := fs.EnsureFileWithPermission(fakeLdconfig, 0755)
	if err != nil {
		t.Fatalf("Could not create fake ldconfig: %s", err)
	}

	pathEnv := os.Getenv("PATH")
	env := os.Environ()
	env = append(env, fmt.Sprintf("PATH=%s:%s", tmpDir, pathEnv))

	// Make sure we fall back to system ldconfig okay
	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("exec"),
		e2e.WithEnv(env),
		e2e.WithArgs("--nv", c.env.ImagePath, "/bin/true"),
		e2e.ExpectExit(0,
			e2e.ExpectError(e2e.ContainMatch, "trying /sbin/ldconfig"),
		),
	)
}
