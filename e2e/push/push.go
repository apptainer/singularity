// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// Package push tests only test the oras transport (and a invalid transport) against a local registry
package push

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/e2e/internal/testhelper"
)

type ctx struct {
	env e2e.TestEnv
}

func (c ctx) testInvalidTransport(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	tests := []struct {
		name       string
		uri        string
		expectOp   e2e.SingularityCmdResultOp
		expectExit int
	}{
		{
			name:       "push invalid transport",
			uri:        "nothing://bar/foo/foobar:latest",
			expectOp:   e2e.ExpectError(e2e.ContainMatch, "Unsupported transport type: nothing"),
			expectExit: 255,
		},
	}

	for _, tt := range tests {
		args := []string{c.env.ImagePath, tt.uri}

		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("push"),
			e2e.WithArgs(args...),
			e2e.ExpectExit(tt.expectExit, tt.expectOp),
		)
	}
}

func (c ctx) testPushCmd(t *testing.T) {
	e2e.EnsureImage(t, c.env)

	e2e.EnsureRegistry(t)

	// setup file and dir to use as invalid sources
	orasInvalidDir, err := ioutil.TempDir(c.env.TestDir, "oras_push_dir-")
	if err != nil {
		err = errors.Wrap(err, "creating oras temporary directory")
		t.Fatalf("unable to create src dir for push tests: %+v", err)
	}

	orasInvalidFile, err := e2e.WriteTempFile(orasInvalidDir, "oras_invalid_image-", "Invalid Image Contents")
	if err != nil {
		err = errors.Wrap(err, "creating oras temporary file")
		t.Fatalf("unable to create src file for push tests: %+v", err)
	}

	tests := []struct {
		desc             string // case description
		dstURI           string // destination URI for image
		imagePath        string // src image path
		expectedExitCode int    // expected exit code for the test
	}{
		{
			desc:             "non existent image",
			imagePath:        filepath.Join(orasInvalidDir, "not_an_existing_file.sif"),
			dstURI:           fmt.Sprintf("oras://%s/non_existent:test", c.env.TestRegistry),
			expectedExitCode: 255,
		},
		{
			desc:             "non SIF file",
			imagePath:        orasInvalidFile,
			dstURI:           fmt.Sprintf("oras://%s/non_sif:test", c.env.TestRegistry),
			expectedExitCode: 255,
		},
		{
			desc:             "directory",
			imagePath:        orasInvalidDir,
			dstURI:           fmt.Sprintf("oras://%s/directory:test", c.env.TestRegistry),
			expectedExitCode: 255,
		},
		{
			desc:             "standard SIF push",
			imagePath:        c.env.ImagePath,
			dstURI:           fmt.Sprintf("oras://%s/standard_sif:test", c.env.TestRegistry),
			expectedExitCode: 0,
		},
	}

	for _, tt := range tests {
		tmpdir, err := ioutil.TempDir(c.env.TestDir, "pull_test.")
		if err != nil {
			t.Fatalf("Failed to create temporary directory for pull test: %+v", err)
		}
		defer os.RemoveAll(tmpdir)

		// We create the list of arguments using a string instead of a slice of
		// strings because using slices of strings most of the type ends up adding
		// an empty elements to the list when passing it to the command, which
		// will create a failure.
		args := tt.dstURI
		if tt.imagePath != "" {
			args = tt.imagePath + " " + args
		}

		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.desc),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("push"),
			e2e.WithArgs(strings.Split(args, " ")...),
			e2e.ExpectExit(tt.expectedExitCode),
		)
	}
}

// E2ETests is the main func to trigger the test suite
func E2ETests(env e2e.TestEnv) testhelper.Tests {
	c := ctx{
		env: env,
	}

	return testhelper.Tests{
		"invalid transport": c.testInvalidTransport,
		"oras":              c.testPushCmd,
	}
}
