// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package version

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"github.com/kelseyhightower/envconfig"
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
		{
			name: "version command",
			args: []string{"version"}
			succeed: true
		},
		{
			name: "version flag",
			args: []string{"--version"}
			succeed: true
		}
	}

	t.Run("verify_version", test.WithoutPrivilege(func(t *testing.T){

		execVersionCmd := exec.Command(testenv.CmdPath, tests[0])

		out, err := execVersionCmd.CombinedOutput()
		if err != nil {
			t.Log(string(out))
			t.Fatalf("Unable to run version command: %v", err)
		}

		execVersionFlag := exec.Command(testenv.CmdPath, tests[1])

		out, err := execVersionFlag.CombinedOutput()
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
