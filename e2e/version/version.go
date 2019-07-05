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
	"github.com/sylabs/singularity/internal/pkg/test/exec"
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
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(c.env.CmdPath, tt.args...)
			res := cmd.Run(t)
			if res.Error != nil {
				t.Fatalf("Failed to obtain version: %+v", res.String())
			}
			outputVersion := strings.TrimPrefix(string(res.Stdout()), "singularity version ")
			outputVersion = strings.TrimSpace(outputVersion)
			if semanticVersion, err := semver.Make(outputVersion); err != nil {
				t.Log(semanticVersion)
				t.Fatalf("FAIL: no semantic version valid for %s command", tt.name)
			}
		})
	}
}

//Test that both versions when running: singularity --version and
// singularity version give the same result
func (c *ctx) testEqualVersion(t *testing.T) {
	var tmpVersion = ""
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(c.env.CmdPath, tt.args...)
			res := cmd.Run(t)
			if res.Error != nil {
				t.Fatalf("Failed to obtain version: %+v", res.String())
			}
			outputVersion := strings.TrimPrefix(string(res.Stdout()), "singularity version ")
			outputVersion = strings.TrimSpace(outputVersion)

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
func RunE2ETests(env e2e.TestEnv) func(*testing.T) {
	c := &ctx{
		env: env,
	}

	return func(t *testing.T) {
		t.Run("test_semantic_version", c.testSemanticVersion)
		t.Run("test_equal_version", c.testEqualVersion)
	}
}
