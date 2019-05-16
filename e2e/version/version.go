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

func verifyVersion(t *testing.T) {

	tests := []struct {
		name    string
		args    []string
		succeed bool
	}{
		{"version command", []string{"version"}, true},
		{"version flag", []string{"--version"}, true},
	}

	t.Run("verify_version", test.WithoutPrivilege(func(t *testing.T) {

		execVersionCmd := exec.Command(testenv.CmdPath, tests[0].args...)
		outputVersionCmd, err := execVersionCmd.CombinedOutput()
		if err != nil {
			t.Log(string(outputVersionCmd))
			t.Fatalf("Unable to run version command: %v", err)
		}
		versionFromVersionCmd, err := semver.Make(string(outputVersionCmd))
		if err != nil {
			t.Fatalf("Unable to obtain semantic version after running version command: %v", err)
		}

		execVersionFlag := exec.Command(testenv.CmdPath, tests[1].args...)
		outputVersionFlag, err := execVersionFlag.CombinedOutput()
		if err != nil {
			t.Log(string(outputVersionFlag))
			t.Fatalf("Unable to run version flag: %v", err)
		}
		versionFromVersionFlag, err := semver.Make(string(outputVersionFlag))

		if err != nil {
			t.Fatalf("Unable to obtain semantic version after running version flag: %v", err)
		}

		if versionFromVersionCmd.Compare(versionFromVersionFlag) != 0 {
			t.Log("FAIL: singularity version command and singularity --version give a non-matching version result")
		} else {
			t.Log("SUCCESS: singularity version command and singularity --version give the same matching version result")
		}

	}))

}

func RunE2ETests(t *testing.T) {
	err := envconfig.Process("E2E", &testenv)
	if err != nil {
		t.Fatal(err.Error())
	}

	t.Run("verify_version", verifyVersion)
}
