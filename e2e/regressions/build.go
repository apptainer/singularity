// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package regressions

import (
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
)

type regressionsTests struct {
	env e2e.TestEnv
}

// This test will build an image from a multi-stage definition
// file, the first stage compile a bad NSS library containing
// a constructor forcing program to exit with code 255 when loaded,
// the second stage will copy the bad NSS library in its root filesytem
// to check that the post section executed by the build engine doesn't
// load the bad NSS library from container image.
// Most if not all NSS services point to the bad NSS library in
// order to catch all the potential calls which could occur from
// Go code inside the build engine, singularity engine is also tested.
func (c *regressionsTests) issue4203(t *testing.T) {
	image := filepath.Join(c.env.TestDir, "issue_4203.sif")

	c.env.RunSingularity(
		t,
		e2e.WithPrivileges(true),
		e2e.WithCommand("build"),
		e2e.WithArgs(image, "testdata/regressions/issue_4203.def"),
		e2e.PostRun(func(t *testing.T) {
			defer os.Remove(image)

			if t.Failed() {
				return
			}

			// also execute the image to check that singularity
			// engine doesn't try to load a NSS library from
			// container image
			c.env.RunSingularity(
				t,
				e2e.WithPrivileges(false),
				e2e.WithCommand("exec"),
				e2e.WithArgs(image, "true"),
				e2e.ExpectExit(0),
			)
		}),
		e2e.ExpectExit(0),
	)
}

// This test will build a sandbox, as a non-root user from a dockerhub image
// that contains a single folder and file with `000` permissions.
// To verify we force files to be accessible / moveable / removable by the user
// we check for `700` and `400` permissions on the folder and file respectively.
func (c *regressionsTests) issue4524(t *testing.T) {
	sandbox := filepath.Join(c.env.TestDir, "issue_4524")

	c.env.RunSingularity(
		t,
		e2e.WithPrivileges(false),
		e2e.WithCommand("build"),
		e2e.WithArgs("--sandbox", sandbox, "docker://sylabsio/issue4524"),
		e2e.PostRun(func(t *testing.T) {

			// If we failed to build the sandbox completely, leave what we have for
			// investigation.
			if t.Failed() {
				t.Logf("Test %s failed, not removing directory %s", t.Name(), sandbox)
				return
			}

			if !e2e.PathPerms(t, path.Join(sandbox, "directory"), 0700) {
				t.Error("Expected 0700 permissions on 000 test directory in rootless sandbox")
			}
			if !e2e.PathPerms(t, path.Join(sandbox, "file"), 0600) {
				t.Error("Expected 0600 permissions on 000 test file in rootless sandbox")
			}

			// If the permissions aren't as we expect them to be, leave what we have for
			// investigation.
			if t.Failed() {
				t.Logf("Test %s failed, not removing directory %s", t.Name(), sandbox)
				return
			}

			err := os.RemoveAll(sandbox)
			if err != nil {
				t.Logf("Cannot remove sandbox directory: %#v", err)
			}

		}),
		e2e.ExpectExit(0),
	)
}

// RunE2ETests is the main func to trigger the test suite
func RunE2ETests(env e2e.TestEnv) func(*testing.T) {
	c := &regressionsTests{
		env: env,
	}

	return func(t *testing.T) {
		t.Run("Issue 4203", c.issue4203) // https://github.com/sylabs/singularity/issues/4203
		t.Run("Issue 4524", c.issue4524) // https://github.com/sylabs/singularity/issues/4524
	}
}
