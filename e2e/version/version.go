// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package version

import (
	"strings"
	"testing"

	"github.com/blang/semver"
	"github.com/pkg/errors"
	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/e2e/internal/testhelper"
)

type ctx struct {
	env e2e.TestEnv
}

var tests = []struct {
	name string
	args []string
}{
	{"version command", []string{"version"}},
	{"version flag", []string{"--version"}},
}

//Test that this version uses the semantic version format
func (c ctx) testSemanticVersion(t *testing.T) {
	for _, tt := range tests {

		checkSemanticVersionFn := func(t *testing.T, r *e2e.SingularityCmdResult) {
			outputVer := strings.TrimPrefix(string(r.Stdout), "singularity version ")
			outputVer = strings.TrimSpace(outputVer)
			if semanticVersion, err := semver.Make(outputVer); err != nil {
				t.Log(semanticVersion)
				t.Errorf("no semantic version valid for %s command", tt.name)
			}
		}

		c.env.RunSingularity(
			t,
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithArgs(tt.args...),
			e2e.PostRun(func(t *testing.T) {
				if t.Failed() {
					t.Log("Failed to obtain version")
				}
			}),
			e2e.ExpectExit(0, checkSemanticVersionFn),
		)
	}
}

//Test that both versions when running: singularity --version and
// singularity version give the same result
func (c ctx) testEqualVersion(t *testing.T) {
	var tmpVersion = ""
	for _, tt := range tests {

		checkEqualVersionFn := func(t *testing.T, r *e2e.SingularityCmdResult) {
			outputVer := strings.TrimPrefix(string(r.Stdout), "singularity version ")
			outputVer = strings.TrimSpace(outputVer)
			semanticVersion, err := semver.Make(outputVer)
			if err != nil {
				err = errors.Wrapf(err, "creating semver version from %q", outputVer)
				t.Fatalf("Creating semver version: %+v", err)
			}
			if tmpVersion != "" {
				versionTmp, err := semver.Make(tmpVersion)
				if err != nil {
					err = errors.Wrapf(err, "creating semver version from %q", tmpVersion)
					t.Fatalf("Creating semver version: %+v", err)
				}
				//compare versions and see if they are equal
				if semanticVersion.Compare(versionTmp) != 0 {
					err = errors.Wrapf(err, "comparing versions %q and %q", outputVer, tmpVersion)
					t.Fatalf("singularity version command and singularity --version give a non-matching version result: %+v", err)
				}
			} else {
				tmpVersion = outputVer
			}
		}

		c.env.RunSingularity(
			t,
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithArgs(tt.args...),
			e2e.PostRun(func(t *testing.T) {
				if t.Failed() {
					t.Log("Failed to obtain version")
				}
			}),
			e2e.ExpectExit(0, checkEqualVersionFn),
		)

	}
}

// Test the help option
func (c ctx) testHelpOption(t *testing.T) {
	c.env.RunSingularity(
		t,
		e2e.WithProfile(e2e.UserProfile),
		e2e.WithCommand("version"),
		e2e.WithArgs("--help"),
		e2e.ExpectExit(
			0,
			e2e.ExpectOutput(e2e.RegexMatch, "^Show the version for Singularity"),
		),
	)
}

// E2ETests is the main func to trigger the test suite
func E2ETests(env e2e.TestEnv) func(*testing.T) {
	c := ctx{
		env: env,
	}

	return testhelper.TestRunner(map[string]func(*testing.T){
		"equal version":    c.testEqualVersion,
		"help option":      c.testHelpOption,
		"semantic version": c.testSemanticVersion,
	})
}
