// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package version

import (
	"strings"
	"testing"

	"github.com/blang/semver"
	"github.com/sylabs/singularity/e2e/internal/e2e"
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
func (c *ctx) testSemanticVersion(t *testing.T) {
	for _, tt := range tests {

		checkSemanticVersionFn := func(t *testing.T, r *e2e.SingularityCmdResult) {
			outputVer := strings.TrimPrefix(string(r.Stdout), "singularity version ")
			outputVer = strings.TrimSpace(outputVer)
			if semanticVersion, err := semver.Make(outputVer); err != nil {
				t.Log(semanticVersion)
				t.Errorf("no semantic version valid for %s command", tt.name)
			}
		}

		e2e.RunSingularity(
			t,
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
func (c *ctx) testEqualVersion(t *testing.T) {
	var tmpVersion = ""
	for _, tt := range tests {

		checkEqualVersionFn := func(t *testing.T, r *e2e.SingularityCmdResult) {
			outputVer := strings.TrimPrefix(string(r.Stdout), "singularity version ")
			outputVer = strings.TrimSpace(outputVer)
			semanticVersion, err := semver.Make(outputVer)
			if err != nil {
				t.Log(semanticVersion)
				t.Fatalf("FAIL: no semantic version valid for %s command", tt.name)
			}
			if tmpVersion != "" {
				versionTmp, err := semver.Make(tmpVersion)
				if err != nil {
					t.Fatalf("FAIL: %s", err)
				}
				//compare versions and see if they are equal
				if semanticVersion.Compare(versionTmp) != 0 {
					t.Fatalf("FAIL: singularity version command and singularity --version give a non-matching version result")
				}
			} else {
				tmpVersion = outputVer
			}
		}

		e2e.RunSingularity(
			t,
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

// RunE2ETests is the main func to trigger the test suite
func RunE2ETests(env e2e.TestEnv) func(*testing.T) {
	c := &ctx{
		env: env,
	}

	return func(t *testing.T) {
		t.Run("test_semantic_version", c.testSemanticVersion)
		t.Run("test_equal_version", c.testEqualVersion)
	}
}
