// Copyright (c) 2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularityenv

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/util/fs"
	"github.com/sylabs/singularity/pkg/util/rlimit"
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
