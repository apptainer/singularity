// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package regressions

import (
	"io/ioutil"
	"log"
	"os"
	"path"
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
// destination directory contains a trailing slash and when it doesn't.
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

				c.env.ImageVerify(t, imagePath, e2e.RootProfile)
			}),
			e2e.ExpectExit(0),
		)
	}
}

// This test will build a sandbox, as a non-root user from a dockerhub image
// that contains a single folder and file with `000` permission.
// It will verify that with `--fix-perms` we force files to be accessible,
// moveable, removable by the user. We check for `700` and `400` permissions on
// the folder and file respectively.
func (c *regressionsTests) issue4524(t *testing.T) {
	sandbox := filepath.Join(c.env.TestDir, "issue_4524")

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("build"),
		e2e.WithArgs("--fix-perms", "--sandbox", sandbox, "docker://sylabsio/issue4524"),
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

func (c *regressionsTests) issue4583(t *testing.T) {
	image := filepath.Join(c.env.TestDir, "issue_4583.sif")

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.RootProfile),
		e2e.WithCommand("build"),
		e2e.WithArgs(image, "testdata/regressions/issue_4583.def"),
		e2e.PostRun(func(t *testing.T) {
			defer os.Remove(image)

			if t.Failed() {
				return
			}
		}),
		e2e.ExpectExit(0),
	)
}

func (c *regressionsTests) issue4943(t *testing.T) {
	const (
		image = "docker://gitlab-registry.cern.ch/linuxsupport/cc7-base:20191107"
	)

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("build"),
		e2e.WithArgs("--force", "/dev/null", image),
		e2e.ExpectExit(0),
	)

}

// Test -c section parameter is correctly handled.
func (c *regressionsTests) issue4967(t *testing.T) {
	image := filepath.Join(c.env.TestDir, "issue_4967.sif")

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.RootProfile),
		e2e.WithCommand("build"),
		e2e.WithArgs(image, "testdata/regressions/issue_4967.def"),
		e2e.PostRun(func(t *testing.T) {
			os.Remove(image)
		}),
		e2e.ExpectExit(
			0,
			e2e.ExpectOutput(e2e.ContainMatch, "function foo"),
		),
	)
}

// The image contains symlinks /etc/resolv.conf and /etc/hosts
// pointing to nowhere, build should pass but with warnings.
func (c *regressionsTests) issue4969(t *testing.T) {
	image := filepath.Join(c.env.TestDir, "issue_4969.sif")

	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.RootProfile),
		e2e.WithCommand("build"),
		e2e.WithArgs(image, "testdata/regressions/issue_4969.def"),
		e2e.PostRun(func(t *testing.T) {
			os.Remove(image)
		}),
		e2e.ExpectExit(
			0,
			e2e.ExpectOutput(e2e.ExactMatch, "TEST OK"),
		),
	)
}

// E2ETests is the main func to trigger the test suite
func E2ETests(env e2e.TestEnv) func(*testing.T) {
	c := regressionsTests{
		env: env,
	}

	return testhelper.TestRunner(map[string]func(*testing.T){
		"issue 4203": c.issue4203, // https://github.com/sylabs/singularity/issues/4203
		"issue 4407": c.issue4407, // https://github.com/sylabs/singularity/issues/4407
		"issue 4524": c.issue4524, // https://github.com/sylabs/singularity/issues/4524
		"issue 4583": c.issue4583, // https://github.com/sylabs/singularity/issues/4583
		"issue 4943": c.issue4943, // https://github.com/sylabs/singularity/issues/4943
		"issue 4967": c.issue4967, // https://github.com/sylabs/singularity/issues/4967
		"issue 4969": c.issue4969, // https://github.com/sylabs/singularity/issues/4969
	})
}
