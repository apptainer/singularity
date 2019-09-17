// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package regressions

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/e2e/internal/testhelper"
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
func (c regressionsTests) issue4203(t *testing.T) {
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

// issue4407 checks that it's possible to build a sandbox image when the
// destination directory contains a traling slash and when it doesn't.
func (c *regressionsTests) issue4407(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	sandboxDir := func() string {
		name, err := ioutil.TempDir(c.env.TestDir, "sandbox.")
		if err != nil {
			log.Fatalf("failed to create temporary directory for sandbox: %v", err)
		}

		if err := os.Chmod(name, 0755); err != nil {
			log.Fatalf("failed to chmod temporary directory for sandbox: %v", err)
		}

		return name
	}

	tc := map[string]string{
		"with slash":    sandboxDir() + "/",
		"without slash": sandboxDir(),
	}

	for name, imagePath := range tc {
		args := []string{
			"--force",
			"--sandbox",
			imagePath,
			c.env.ImagePath,
		}

		c.env.RunSingularity(
			t,
			e2e.AsSubtest(name),
			e2e.WithProfile(e2e.RootProfile),
			e2e.WithCommand("build"),
			e2e.WithArgs(args...),
			e2e.PostRun(func(t *testing.T) {
				if t.Failed() {
					return
				}

				defer os.RemoveAll(imagePath)

				c.env.ImageVerify(t, imagePath)
			}),
			e2e.ExpectExit(0),
		)
	}
}

// E2ETests is the main func to trigger the test suite
func E2ETests(env e2e.TestEnv) func(*testing.T) {
	c := regressionsTests{
		env: env,
	}

	return testhelper.TestRunner(map[string]func(*testing.T){
		"issue 4203": c.issue4203, // https://github.com/sylabs/singularity/issues/4203
		"issue 4407": c.issue4407, // https://github.com/sylabs/singularity/issues/4407
	})
}
