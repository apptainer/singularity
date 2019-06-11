// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package keypublic

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kelseyhightower/envconfig"
	"github.com/sylabs/singularity/e2e/internal/keyexec"
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
		succeed bool
	}{
		// TODO: Uncomment when Travis/CCI has a access token
		/*		{
					name:    "push test key",
					args:    []string{"push", "F69C21F759C8EA06FD32CCF4536523CE1E109AF3"},
					succeed: true,
				},
				{
					name:    "push test key fail",
					args:    []string{"push", "F69C21F759C8EA06FD32CCF4536523CE1E109AF3Z"},
					succeed: false,
				},*/
		{
			name:    "search key",
			args:    []string{"search", "e2e test key"},
			succeed: true,
		},
		{
			name:    "search key",
			args:    []string{"search", "e2e"},
			succeed: true,
		},
		{
			name:    "search key id",
			args:    []string{"search", "0x1E109AF3"},
			succeed: true,
		},
		{
			name:    "search key no key",
			args:    []string{"search", "0xRANDOME123"},
			succeed: false,
		},
	}

	test.WithoutPrivilege(func(t *testing.T) {
		c, b, err := keyexec.ImportKey(t, defaultKeyFile)
		if err != nil {
			t.Log("command that failed: ", c, string(b))
			t.Fatalf("Unable to import key: %v", err)
		}
	})

	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			cmd, out, err := keyexec.RunKeyCmd(t, testenv.CmdPath, tt.args, "", "")
			if tt.succeed {
				if err != nil {
					t.Log("Command that failed: ", cmd)
					t.Log(string(out))
					t.Fatalf("Unexpected failure: %v", err)
				}
			} else {
				if err == nil {
					t.Log(string(out))
					t.Fatalf("Unexpected success when running: %s", cmd)
				}
			}
		}))
	}
}

func testPublicKeyImportExport(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		stdin   string
		file    string
		succeed bool
	}{
		{
			name:    "export public",
			args:    []string{"export"},
			stdin:   "0\n", // TODO: this will need to be '1' at some point in time -> issue #3199
			file:    defaultKeyFile,
			succeed: true,
		},
		{
			name:    "export public armor",
			args:    []string{"export", "--armor"},
			stdin:   "0\n",
			file:    defaultKeyFile,
			succeed: true,
		},
		{
			name:    "export public armor panic",
			args:    []string{"export", "--armor"},
			stdin:   "1\n",
			file:    defaultKeyFile,
			succeed: false,
		},
		{
			name:    "export armor invalid",
			args:    []string{"export", "--armor"},
			stdin:   "n\n",
			file:    defaultKeyFile,
			succeed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			os.RemoveAll(filepath.Join(keyPath, defaultKeyFile))
			cmd, out, err := keyexec.RunKeyCmd(t, testenv.CmdPath, tt.args, tt.file, tt.stdin)
			if tt.succeed {
				if err != nil {
					t.Log("Command that failed: ", cmd)
					t.Log(string(out))
					t.Fatalf("Unexpected failure: %v", err)
				}

				t.Run("remove_public_key_before_importing", test.WithoutPrivilege(func(t *testing.T) { keyexec.RemoveDefaultPublicKey(t) }))
				t.Run("import_public_key_from", test.WithoutPrivilege(func(t *testing.T) {
					c, b, err := keyexec.ImportKey(t, defaultKeyFile)
					if err != nil {
						t.Log("command that failed: ", c, string(b))
						t.Fatalf("Unable to import key: %v", err)
					}
				}))
			} else {
				if err == nil {
					t.Log(string(out))
					t.Fatalf("Unexpected success when running: %s", cmd)
				}
			}
		}))
	}
}

func TestAll(t *testing.T) {
	err := envconfig.Process("E2E", &testenv)
	if err != nil {
		t.Fatal(err.Error())
	}

	keyPath = testenv.TestDir
	defaultKeyFile = filepath.Join(keyPath, "exported_key")

	// Pull the default public key
	t.Run("pull_default_key", test.WithoutPrivilege(func(t *testing.T) { keyexec.PullDefaultPublicKey(t) }))

	// Run the tests
	t.Run("push_search", testPublicKey)
	t.Run("pubic_key", testPublicKeyImportExport)
}
