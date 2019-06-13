// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// Package push tests only test the oras transport against a local registry
package push

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/internal/pkg/test"
)

type testingEnv struct {
	CmdPath     string `split_words:"true"`
	TestDir     string `split_words:"true"`
	ImagePath   string `split_words:"true"`
	RunDisabled bool   `default:"false"`
}

var testenv testingEnv

func testPushCmd(t *testing.T) {

	// setup file and dir to use as invalid sources
	orasInvalidDir, err := ioutil.TempDir(testenv.TestDir, "oras_push_dir-")
	if err != nil {
		t.Fatalf("unable to create src dir for push tests: %v", err)
	}

	orasInvalidFile, err := e2e.WriteTempFile(orasInvalidDir, "oras_invalid_image-", "Invalid Image Contents")
	if err != nil {
		t.Fatalf("unable to create src file for push tests: %v", err)
	}

	tests := []struct {
		desc          string // case description
		dstURI        string // destination URI for image
		imagePath     string // src image path
		expectSuccess bool   // singularity should exit with code 0
	}{
		{
			desc:          "non existent image",
			imagePath:     filepath.Join(orasInvalidDir, "not_an_existing_file.sif"),
			dstURI:        "oras://localhost:5000/non_existent:test",
			expectSuccess: false,
		},
		{
			desc:          "non SIF file",
			imagePath:     orasInvalidFile,
			dstURI:        "oras://localhost:5000/non_sif:test",
			expectSuccess: false,
		},
		{
			desc:          "directory",
			imagePath:     orasInvalidDir,
			dstURI:        "oras://localhost:5000/directory:test",
			expectSuccess: false,
		},
		{
			desc:          "standard SIF push",
			imagePath:     testenv.ImagePath,
			dstURI:        "oras://localhost:5000/standard_sif:test",
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, test.WithoutPrivilege(func(t *testing.T) {
			tmpdir, err := ioutil.TempDir(testenv.TestDir, "pull_test.")
			if err != nil {
				t.Fatalf("Failed to create temporary directory for pull test: %+v", err)
			}
			defer os.RemoveAll(tmpdir)

			cmd, out, err := e2e.ImagePush(t, tt.imagePath, tt.dstURI)
			switch {
			case tt.expectSuccess && err == nil:
				// PASS: expecting success, succeeded

			case !tt.expectSuccess && err != nil:
				// PASS: expecting failure, failed

			case tt.expectSuccess && err != nil:
				// FAIL: expecting success, failed

				t.Logf("Running command:\n%s\nOutput:\n%s\n", cmd, out)
				t.Errorf("unexpected failure: %v", err)

			case !tt.expectSuccess && err == nil:
				// FAIL: expecting failure, succeeded

				t.Logf("Running command:\n%s\nOutput:\n%s\n", cmd, out)
				t.Errorf("unexpected success: command should have failed")
			}
		}))
	}
}

// RunE2ETests is the main func to trigger the test suite
func RunE2ETests(t *testing.T) {
	e2e.LoadEnv(t, &testenv)
	e2e.EnsureImage(t)

	t.Run("push", testPushCmd)
}
