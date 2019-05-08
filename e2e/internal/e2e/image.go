// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package e2e

import (
	"os"
	"os/exec"
	"testing"
)

var testenv = struct {
	CmdPath   string `split_words:"true"` // singularity program
	ImagePath string `split_words:"true"` // base image for tests
}{}

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

	opts := BuildOpts{
		Force:   true,
		Sandbox: false,
	}

	b, err := ImageBuild(
		testenv.CmdPath,
		opts,
		testenv.ImagePath,
		"./testdata/Singularity")

	if err != nil {
		t.Logf("Failed to build image %q.\nOutput:\n%s\n",
			testenv.ImagePath,
			b)
		t.Fatalf("Unexpected failure: %+v", err)
	}
}

// PullTestAlpineContainer will pull the 'library://alpine:latest' container for tests.
// This will pull to the pervided path ('imagePath'), and overide any image that was there.
func PullTestAlpineContainer(cmdPath string, imagePath string) ([]byte, error) {
	argv := []string{"pull", "--allow-unsigned", "--force", imagePath, "library://alpine:latest"}
	cmd := exec.Command(cmdPath, argv...)

	return cmd.CombinedOutput()
}
