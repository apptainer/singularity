// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package e2e

import (
	"os"
	"testing"
)

var testenv = struct {
	CmdPath   string `split_words:"true"` // singularity program
	ImagePath string `split_words:"true"` // base image for tests
}{}

// EnsureImage checks if e2e test image is already built or built
// it otherwise.
func EnsureImage(t *testing.T) {
	LoadEnv(t, &testenv)

	switch _, err := os.Stat(testenv.ImagePath); {
	case err == nil:
		// OK: file exists, return
		return

	case os.IsNotExist(err):
		// OK: file does not exist, continue

	default:
		// FATAL: something else is wrong
		t.Fatalf("Failed when checking image %q: %+v\n",
			testenv.ImagePath,
			err)
	}

	RunSingularity(
		t,
		"BuildTestImage",
		WithoutSubTest(),
		WithPrivileges(true),
		WithCommand("build"),
		WithArgs("--force", testenv.ImagePath, "testdata/Singularity"),
		ExpectExit(0),
	)
}
