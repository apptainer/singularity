// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package keypublic

import (
	"fmt"
	"os/exec"
	"strings"
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
var keyPath string

func runKey(t *testing.T, commands []string, file string) (string, []byte, error) {
	argv := []string{"key"}

	argv = append(argv, commands...)

	if file != "" {
		argv = append(argv, file)
	}

	cmd := fmt.Sprintf("%s %s", testenv.CmdPath, strings.Join(argv, " "))
	out, err := exec.Command(testenv.CmdPath, argv...).CombinedOutput()

	t.Log("???????EXEC_COMMAND: ", cmd)
	t.Log("???????OUTPUT: ", string(out))

	return cmd, out, err
}

func testPublicKey(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		file    string
		succeed bool
	}{
		// taget UID/GID
		{
			name:    "key_list",
			args:    []string{"list"},
			file:    "",
			succeed: true,
		},
	}

	for _, tt := range tests {
		t.Run("key_run "+tt.name, test.WithoutPrivilege(func(t *testing.T) {
			cmd, out, err := runKey(t, tt.args, tt.file)
			if tt.succeed {
				if err != nil {
					t.Log("Command that failed: ", cmd)
					t.Log(string(out))
					t.Fatalf("Unexpected failure: %v", err)
				}
			} else {
				if err == nil {
					t.Log("Command that succeed: ", cmd)
					t.Log(string(out))
					t.Fatalf("Unexpected success: %v", err)
				}
			}
		}))
	}
}

// TestAll is trigered by ../key.go, that is trigered by suite.go in the e3e test directory
func TestAll(t *testing.T) {
	err := envconfig.Process("E2E", &testenv)
	if err != nil {
		t.Fatal(err.Error())
	}

	keyPath = testenv.TestDir

	t.Log("######TMP_PATH: ", keyPath)

	t.Run("pubic_key", testPublicKey)
}
