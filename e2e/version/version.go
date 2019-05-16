// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package version

import (
	"os/exec"
	"testing"

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

func verifyVersion(t *testing.T){
	
	tests := []struct{
		name string
		args []string
		succeed bool
	}{
		{"version command", []string{"version"}, true, },
		{"version flag", []string{"--version"}, true },
	}

	t.Run("verify_version", test.WithoutPrivilege(func(t *testing.T){

		execVersionCmd := exec.Command(testenv.CmdPath, tests[0].args...)

		out, err := execVersionCmd.CombinedOutput()
		if err != nil {
			t.Log(string(out))
			t.Fatalf("Unable to run version command: %v", err)
		}

		execVersionFlag := exec.Command(testenv.CmdPath, tests[1].args...)

		out, err = execVersionFlag.CombinedOutput()
		if err != nil {
			t.Log(string(out))
			t.Fatalf("Unable to run version command: %v", err)
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
