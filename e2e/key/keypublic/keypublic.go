// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package keypublic

import (
	//	"fmt"
	//	"os/exec"
	//	"strings"
	"os"
	"path/filepath"
	"testing"

	"github.com/kelseyhightower/envconfig"
	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/internal/pkg/test"
)

type testingEnv struct {
	// base env for running tests
	CmdPath     string `split_words:"true"`
	TestDir     string `split_words:"true"`
	RunDisabled bool   `default:"false"`
}

const defaultKeyFile = "exported_key"

var testenv testingEnv
var keyPath string

func testPublicKey(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		file    string
		succeed bool
	}{
		{
			name:    "export_public",
			args:    []string{"export"},
			file:    defaultKeyFile,
			succeed: true,
		},
		{
			name:    "key_list_secret",
			args:    []string{"list", "-s"},
			file:    "",
			succeed: true,
		},
		{
			name:    "key_list_bad_flag",
			args:    []string{"list", "--not-a-flag"},
			file:    "",
			succeed: false,
		},
		{
			name:    "key_bad_cmd",
			args:    []string{"notacmd"},
			file:    "",
			succeed: false,
		},
		{
			name:    "key_bad_cmd_flag",
			args:    []string{"notacmd", "--bad"},
			file:    "",
			succeed: false,
		},
		{
			name:    "key_flag",
			args:    []string{"--notaflag"},
			file:    "",
			succeed: false,
		},
	}

	for _, tt := range tests {
		t.Run("key_run "+tt.name, test.WithoutPrivilege(func(t *testing.T) {
			os.RemoveAll(filepath.Join(keyPath, defaultKeyFile))
			cmd, out, err := e2e.RunKeyCmd(t, testenv.CmdPath, tt.args, tt.file, "1\n")
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
