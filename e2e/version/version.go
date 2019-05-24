// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package version

import (
	"os/exec"
	"testing"

	"github.com/blang/semver"
	"github.com/kelseyhightower/envconfig"
	"github.com/sylabs/singularity/internal/pkg/test"
)

type testingEnv struct {
	// base env for running tests
	CmdPath     string `split_words:"true"`
	TestDir     string `split_words:"true"`
	RunDisabled bool   `default:"false"`
}

var testenv testingEnv
var tests = []struct {
	name string
	args []string
}{
	{"version command", []string{"version"}},
	{"version flag", []string{"--version"}},
}

//Test that this version uses the semantic version format
func testSemanticVersion(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			out, err := exec.Command(testenv.CmdPath, tt.args...).CombinedOutput()
			if err != nil {
				t.Fatalf("Failed to obtain version: %+v", err)
			}

			if semanticVersion, err := semver.Make(string(out)); err != nil {
				t.Log(semanticVersion)
				t.Fatalf("FAIL: no semantic version valid for %s command", tt.name)
			}
		}))
	}
}

//Test that both versions when running: singularity --version and
// singularity version give the same result
func testEqualVersion(t *testing.T) {
	var tmpVersion = ""
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := exec.Command(testenv.CmdPath, tt.args...).CombinedOutput()
			if semanticVersion, err := semver.Make(string(out)); err != nil {
				t.Log(semanticVersion)
				t.Fatalf("FAIL: no semantic version valid for %s command", tt.name)
			}

			tmpVersion = string(out)
			versionOutput, err := semver.Make(string(out))
			if err != nil {
				t.Fatalf("FAIL: %s", err)
			}
			versionTmp, err := semver.Make(tmpVersion)
			if err != nil {
				t.Fatalf("FAIL: %s", err)
			}

			if tmpVersion != "" {
				//compare versions and see if they are equal
				if versionOutput.Compare(versionTmp) != 0 {
					t.Fatalf("FAIL: singularity version command and singularity --version give a non-matching version result")
				}
			}
		})
	}
}

// RunE2ETests is the main func to trigger the test suite
func RunE2ETests(t *testing.T) {
	err := envconfig.Process("E2E", &testenv)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("test_semantic_version", testSemanticVersion)
	t.Run("test_equal_version", testEqualVersion)
}
