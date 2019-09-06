// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package regressions

import (
	"os"
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
		e2e.WithProfile(e2e.RootProfile),
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
				e2e.WithProfile(e2e.UserProfile),
				e2e.WithCommand("exec"),
				e2e.WithArgs(image, "true"),
				e2e.ExpectExit(0),
			)
		}),
		e2e.ExpectExit(0),
	)
}

// E2ETests is the main func to trigger the test suite
func E2ETests(env e2e.TestEnv) func(*testing.T) {
	c := &regressionsTests{
		env: env,
	}

	return func(t *testing.T) {
		// https://github.com/sylabs/singularity/issues/4203
		t.Run("Issue4203", c.issue4203)
	}
}
