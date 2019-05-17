// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package version

import (
	"github.com/blang/semver"
	"github.com/kelseyhightower/envconfig"
	"github.com/sylabs/singularity/internal/pkg/test"
	"testing"
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

func testSemanticVersion(t *testing.T) {

	c := test.NewCmd(testenv.CmdPath)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := c.Run(t, tt.args...).Stdout()
			if semanticVersion, err := semver.Make(out); err != nil {
				t.Log(semanticVersion)
				t.Logf("FAIL: no semantic version valid for %s command", tt.name)
			}
		})
	}
}

func testEqualVersion(t *testing.T) {

	c := test.NewCmd(testenv.CmdPath)
	var tmpVersion = ""
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := c.Run(t, tt.args...).Stdout()
			if semanticVersion, err := semver.Make(out); err != nil {
				t.Log(semanticVersion)
				t.Logf("FAIL: no semantic version valid for %s command", tt.name)
			}

			tmpVersion = out
			versionOutput, err := semver.Make(out)
			if err != nil {
				t.Logf("FAIL: %s", err)
			}
			versionTmp, err := semver.Make(tmpVersion)
			if err != nil {
				t.Logf("FAIL: %s", err)
			}

			if tmpVersion != "" {
				//compare versions and see if they are equal
				if versionOutput.Compare(versionTmp) != 0 {
					t.Log("FAIL: singularity version command and singularity --version give a non-matching version result")
				} else {
					t.Log("SUCCESS: singularity version command and singularity --version give the same matching version result")
				}
			}
		})
	}
}

func RunE2ETests(t *testing.T) {
	err := envconfig.Process("E2E", &testenv)
	if err != nil {
		t.Fatal(err.Error())
	}

	t.Run("test_semantic_version", testSemanticVersion)
	t.Run("test_equal_version", testEqualVersion)
}
