// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package keyprivate

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
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

// corruptKey will take a ASCII key (kpath) and change some chars in it (corrupt it).
func corruptKey(t *testing.T, kpath string) {
	input, err := ioutil.ReadFile(kpath)
	if err != nil {
		t.Fatalf("Unable to read file: %v", err)
	}

	lines := strings.Split(string(input), "\n")

	for i, line := range lines {
		if strings.Contains(line, "B") {
			lines[i] = "P"
		}
	}
	output := strings.Join(lines, "\n")
	err = ioutil.WriteFile(kpath, []byte(output), 0644)
	if err != nil {
		t.Fatalf("Unable to write to file: %v", err)
	}
}

func testPrivateKey(t *testing.T) {
	tests := []struct {
		name    string
		armor   bool
		stdin   int
		file    string
		corrupt bool
		succeed bool
	}{
		{
			name:    "export private",
			armor:   false,
			stdin:   0, // TODO: this will need to be '1' at some point in time -> issue #3199
			file:    defaultKeyFile,
			succeed: true,
		},
		{
			name:    "export private armor",
			armor:   true,
			stdin:   0, // TODO: this will need to be '1' at some point in time -> issue #3199
			file:    defaultKeyFile,
			succeed: true,
		},
		{
			name:    "export private armor corrupt",
			armor:   true,
			stdin:   0, // TODO: this will need to be '1' at some point in time -> issue #3199
			file:    defaultKeyFile,
			corrupt: true,
			succeed: false,
		},
		{
			name:    "export private panic",
			armor:   false,
			stdin:   1, // TODO: this will need to be '1' at some point in time -> issue #3199
			file:    defaultKeyFile,
			succeed: false,
		},
		{
			name:    "export private armor panic",
			armor:   true,
			stdin:   1, // TODO: this will need to be '1' at some point in time -> issue #3199
			file:    defaultKeyFile,
			succeed: false,
		},
	}

	for _, tt := range tests {
		t.Run("key_run", test.WithoutPrivilege(func(t *testing.T) {
			os.RemoveAll(filepath.Join(defaultKeyFile))
			out, err := e2e.ExportPrivateKey(t, tt.file, tt.stdin, tt.armor)
			if tt.succeed {
				if err != nil {
					t.Log(string(out))
					t.Fatalf("Unexpected failure: %v", err)
				}

				t.Run("remove_private_keyring_before_importing", test.WithoutPrivilege(func(t *testing.T) { e2e.RemoveSecretKeyring(t) }))
				t.Run("import_private_keyring_from", test.WithoutPrivilege(func(t *testing.T) {
					b, err := e2e.ImportPrivateKey(t, defaultKeyFile)
					if err != nil {
						t.Log(string(b))
						t.Fatalf("Unable to import key: %v", err)
					}
				}))
			} else {
				// if the test key is corrupted, try to import it, should fail
				if tt.corrupt {
					t.Run("corrupting_key", test.WithoutPrivilege(func(t *testing.T) { corruptKey(t, defaultKeyFile) }))
					t.Run("import_private_key", test.WithoutPrivilege(func(t *testing.T) {
						b, err := e2e.ImportKey(t, defaultKeyFile)
						if err == nil {
							t.Fatalf("Unexpected success: %s", string(b))
						}
					}))
				} else {
					if err == nil {
						t.Log(string(out))
						t.Fatalf("Unexpected success")
					}
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

	t.Run("backingup_secret_keyring", test.WithoutPrivilege(func(t *testing.T) { e2e.BackupSecretKeyring(t) }))
	t.Run("importing_test_key", test.WithoutPrivilege(func(t *testing.T) {
		b, err := e2e.ImportPrivateKey(t, "./key/testdata/e2e_test_key.asc")
		if err != nil {
			t.Log(string(b))
			t.Fatalf("Unable to import test key: %v", err)
		}
	}))

	keyPath = testenv.TestDir
	defaultKeyFile = filepath.Join(keyPath, "exported_private_key")

	// Pull the default public key
	t.Run("pull_default_key", test.WithoutPrivilege(func(t *testing.T) { e2e.PullDefaultPublicKey(t) }))

	// Run the tests
	t.Run("public_key", testPrivateKey)

	// Recover the secret keyring
	t.Run("recovering_secret_keyring", test.WithoutPrivilege(func(t *testing.T) { e2e.RecoverSecretKeyring(t) }))
}