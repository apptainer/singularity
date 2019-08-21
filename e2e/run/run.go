// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package run

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
)

type ctx struct {
	env e2e.TestEnv
}

// testRun555Cache tests the specific case where the cache directory is
// 0555 for access rights, and we try to run a singularity run command
// using that directory as cache. This reflects a problem that is important
// for the grid use case.
func (c *ctx) testRun555Cache(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "e2e-run-555-")
	if err != nil {
		t.Fatalf("failed to create temporary directory: %s", err)
	}
	defer func() {
		err := os.RemoveAll(tempDir)
		if err != nil {
			t.Fatalf("failed to delete temporary directory %s: %s", tempDir, err)
		}
	}()
	cacheDir := filepath.Join(tempDir, "image-cache")
	err = os.Mkdir(cacheDir, 0555)
	if err != nil {
		t.Fatalf("failed to create a temporary image cache: %s", err)
	}
	// Directory is deleted when tempDir is deleted

	cmdArgs := []string{"library://godlovedc/funny/lolcow"}
	c.env.ImgCacheDir = cacheDir
	c.env.RunSingularity(
		t,
		e2e.WithPrivileges(false),
		e2e.WithCommand("run"),
		e2e.WithArgs(cmdArgs...),
		e2e.ExpectExit(0),
	)
}

// RunE2ETests is the main func to trigger the test suite
func CmdE2ETests(env e2e.TestEnv) func(*testing.T) {
	c := &ctx{
		env: env,
	}

	return func(t *testing.T) {
		t.Run("run555cache", c.testRun555Cache)
	}
}
