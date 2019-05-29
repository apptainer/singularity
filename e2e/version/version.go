// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package version

import (
	"os/exec"
	"strings"
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
			outputVersion := strings.TrimPrefix(string(out), "singularity version ")
			outputIndex := strings.Index(outputVersion, "-")
			outputVersion = outputVersion[:outputIndex]
			if semanticVersion, err := semver.Make(outputVersion); err != nil {
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
			if err != nil {
				t.Log(string(out))
				t.Fatalf("Failed to obtain version: %+v", err)
			}
			outputVersion := strings.TrimPrefix(string(out), "singularity version ")
			outputIndex := strings.Index(outputVersion, "-")
			outputVersion = outputVersion[:outputIndex]

			semanticVersion, err := semver.Make(string(outputVersion))
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
				tmpVersion = outputVersion
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
