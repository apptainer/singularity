// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package key

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
)

type ctx struct {
	env                    e2e.TestEnv
	publicExportPath       string
	publicExportASCIIPath  string
	privateExportPath      string
	privateExportASCIIPath string
	keyRing                string
}

func (c *ctx) singularityKeyList(t *testing.T) {
	printOut := func(t *testing.T, r *e2e.SingularityCmdResult) {
		t.Log("stdout from 'key list' : ", string(r.Stdout))
	}

	c.env.RunSingularity(
		t,
		e2e.WithCommand("key"),
		e2e.WithArgs("list"),
		e2e.WithSypgpDir(c.keyRing),
		e2e.ExpectExit(0, printOut),
	)
}

func (c *ctx) singularityKeyNewpair(t *testing.T) {
	tests := []struct {
		name       string
		consoleOps []e2e.SingularityConsoleOp
	}{
		{
			name: "newpair",
			consoleOps: []e2e.SingularityConsoleOp{
				e2e.ConsoleSendLine("e2e test key"),         // Name of the key
				e2e.ConsoleSendLine("westley@sylabs.io"),    // Email for the key
				e2e.ConsoleSendLine("Only for E2E testing"), // Optional comment
				e2e.ConsoleSendLine("e2etests"),             // Password
				e2e.ConsoleSendLine("e2etests"),             // Password confirmation
				e2e.ConsoleSendLine("n"),                    // 'n' to NOT push to the keystore
			},
		},
	}

	for _, tt := range tests {
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.ConsoleRun(tt.consoleOps...),
			e2e.WithCommand("key"),
			e2e.WithArgs("newpair"),
			e2e.WithSypgpDir(c.keyRing),
			e2e.ExpectExit(0),
		)
	}
}

// singularityKeyExport will export a private, and public (binary and ASCII) key.
func (c *ctx) singularityKeyExport(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		consoleOps []e2e.SingularityConsoleOp
	}{
		{
			name: "export public binary",
			args: []string{"export", c.publicExportPath},
			consoleOps: []e2e.SingularityConsoleOp{
				e2e.ConsoleSendLine("0"),
			},
		},
		{
			name: "export private binary",
			args: []string{"export", "--secret", c.privateExportPath},
			consoleOps: []e2e.SingularityConsoleOp{
				e2e.ConsoleSendLine("0"),
				e2e.ConsoleSendLine("e2etests"),
			},
		},
		{
			name: "export public ascii",
			args: []string{"export", "--armor", c.publicExportASCIIPath},
			consoleOps: []e2e.SingularityConsoleOp{
				e2e.ConsoleSendLine("0"),
			},
		},
		{
			name: "export private ascii",
			args: []string{"export", "--secret", "--armor", c.privateExportASCIIPath},
			consoleOps: []e2e.SingularityConsoleOp{
				e2e.ConsoleSendLine("0"),
				e2e.ConsoleSendLine("e2etests"),
			},
		},
	}

	for _, tt := range tests {
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithCommand("key"),
			e2e.WithArgs(tt.args...),
			e2e.ConsoleRun(tt.consoleOps...),
			e2e.WithSypgpDir(c.keyRing),
			e2e.ExpectExit(0),
		)
	}
}

// singularityKeyImport will export a private, and public (binary and ASCII) key.
// And will try (and fail) to import a key with the wrong password.
func (c *ctx) singularityKeyImport(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		consoleOps []e2e.SingularityConsoleOp
		expectExit int
	}{
		{
			name:       "import public binary",
			args:       []string{"import", c.publicExportPath},
			expectExit: 0,
		},
		{
			name: "import private binary wrong password",
			args: []string{"import", c.privateExportPath},
			consoleOps: []e2e.SingularityConsoleOp{
				e2e.ConsoleSendLine("theWrongPassword"), // The wrong password to decrypt the key (will fail)
				e2e.ConsoleSendLine("somethingElse"),
				e2e.ConsoleSendLine("somethingElse"),
			},
			expectExit: 2,
		},
		{
			name: "import private binary",
			args: []string{"import", c.privateExportPath},
			consoleOps: []e2e.SingularityConsoleOp{
				e2e.ConsoleSendLine("e2etests"), // The password to decrypt the key
				e2e.ConsoleSendLine("e2etests"), // Then the new password
				e2e.ConsoleSendLine("e2etests"), // Confirm the password
			},
			expectExit: 0,
		},
		{
			name:       "import public ascii",
			args:       []string{"import", c.publicExportASCIIPath},
			expectExit: 0,
		},
		{
			name: "import private ascii wrong password",
			args: []string{"import", c.privateExportASCIIPath},
			consoleOps: []e2e.SingularityConsoleOp{
				e2e.ConsoleSendLine("theWrongPassword"), // The wrong password to decrypt the key (will fail)
				e2e.ConsoleSendLine("somethingElse"),
				e2e.ConsoleSendLine("somethingElse"),
			},
			expectExit: 2,
		},
		{
			name: "import private ascii",
			args: []string{"import", c.privateExportASCIIPath},
			consoleOps: []e2e.SingularityConsoleOp{
				e2e.ConsoleSendLine("e2etests"), // The password to decrypt the key
				e2e.ConsoleSendLine("e2etests"), // Then the new password
				e2e.ConsoleSendLine("e2etests"), // Confirm the password
			},
			expectExit: 0,
		},
	}

	for _, tt := range tests {
		c.singularityResetKeyring(t) // Remove the tmp keyring before each import
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithCommand("key"),
			e2e.WithArgs(tt.args...),
			e2e.ConsoleRun(tt.consoleOps...),
			e2e.WithSypgpDir(c.keyRing),
			e2e.ExpectExit(tt.expectExit),
		)
	}
}

func (c *ctx) singularityResetKeyring(t *testing.T) {
	// TODO: run this as non-root
	err := os.RemoveAll(c.keyRing)
	if os.IsNotExist(err) && err != nil {
		t.Fatalf("unable to remove tmp keyring directory: %s", err)
	}
}

// Run the 'key' tests in order
func (c *ctx) singularityKeyCmd(t *testing.T) {
	c.singularityKeyNewpair(t)
	c.singularityKeyExport(t)
	c.singularityKeyImport(t)
	c.singularityKeyExport(t)
	c.singularityKeyImport(t)
	c.singularityKeyList(t)
}

// RunE2ETests is the main func to trigger the test suite
func RunE2ETests(env e2e.TestEnv) func(*testing.T) {
	c := &ctx{
		env:                    env,
		publicExportPath:       filepath.Join(env.TestDir, "public_key.asc"),
		publicExportASCIIPath:  filepath.Join(env.TestDir, "public_ascii_key.asc"),
		privateExportPath:      filepath.Join(env.TestDir, "private_key.asc"),
		privateExportASCIIPath: filepath.Join(env.TestDir, "private_ascii_key.asc"),
		keyRing:                filepath.Join(env.TestDir, "sypgp-test-keyring"),
	}

	if err := os.Setenv("SINGULARITY_SYPGPDIR", c.keyRing); err != nil {
		panic(fmt.Sprintf("unable to set keyring: %s", err))
	}

	return func(t *testing.T) {
		t.Run("keyCmd", c.singularityKeyCmd) // Run all the tests in order
	}
}
