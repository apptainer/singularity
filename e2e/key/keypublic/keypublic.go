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

var testenv testingEnv
var keyPath string
var defaultKeyFile string

func testPublicKey(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		stdin   string
		file    string
		succeed bool
	}{
		{
			name:    "export_public",
			args:    []string{"export"},
			stdin:   "0\n",
			file:    defaultKeyFile,
			succeed: true,
		},
		{
			name:    "export_public_armor",
			args:    []string{"export", "--armor"},
			stdin:   "0\n",
			file:    defaultKeyFile,
			succeed: true,
		},
		{
			name:    "export_public_armor_panic",
			args:    []string{"export", "--armor"},
			stdin:   "1\n",
			file:    defaultKeyFile,
			succeed: false,
		},
	}

	for _, tt := range tests {
		t.Run("key_run "+tt.name, test.WithoutPrivilege(func(t *testing.T) {
			os.RemoveAll(filepath.Join(keyPath, defaultKeyFile))
			cmd, out, err := e2e.RunKeyCmd(t, testenv.CmdPath, tt.args, tt.file, tt.stdin)
			if tt.succeed {
				if err != nil {
					t.Log("Command that failed: ", cmd)
					t.Log(string(out))
					t.Fatalf("Unexpected failure: %v", err)
				}
				t.Run("remove_public_key_before_importing", test.WithoutPrivilege(func(t *testing.T) { e2e.ImportKey(t, filepath.Join(keyPath, defaultKeyFile)) }))
				t.Run("import_public_key_from:"+tt.name, test.WithoutPrivilege(func(t *testing.T) { e2e.ImportKey(t, filepath.Join(keyPath, defaultKeyFile)) }))
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
	defaultKeyFile = filepath.Join(keyPath, "exported_key")

	// Pull the default public key
	t.Run("pull_default_key", test.WithoutPrivilege(func(t *testing.T) { e2e.PullDefaultPublicKey(t) }))

	t.Run("pubic_key", testPublicKey)

	//t.Run("remove_default_key", test.WithoutPrivilege(func(t *testing.T) {e2e.RemoveDefaultPublicKey(t)}))
}
