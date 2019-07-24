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
		e2e.WithPrivileges(false),
		e2e.WithCommand("key"),
		e2e.WithArgs("list"),
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
				e2e.ConsoleSendLine("e2e test key; do not use"),
				e2e.ConsoleSendLine("westley@sylabs.io"),
				e2e.ConsoleSendLine("E2E test key; do not use"),
				e2e.ConsoleSendLine("e2etests"),
				e2e.ConsoleSendLine("e2etests"),
				e2e.ConsoleSendLine("n"),
			},
		},
	}

	for _, tt := range tests {
		c.env.RunSingularity(
			t,
			e2e.WithPrivileges(false),
			e2e.ConsoleRun(tt.consoleOps...),
			e2e.WithCommand("key"),
			e2e.WithArgs("newpair"),
			e2e.ExpectExit(0),
		)
	}
	//t.Run("singularityKeyExport", c.singularityKeyExport)
}

// singularityKeyExport will export a private, and public (binary and ASCII) key.
func (c *ctx) singularityKeyExport(t *testing.T) {
	tests := []struct {
		name       string
		armor      bool
		secret     bool
		exportPath string
		consoleOps []e2e.SingularityConsoleOp
	}{
		{
			name:       "export public binary",
			exportPath: c.publicExportPath,
			armor:      false,
			secret:     false,
			consoleOps: []e2e.SingularityConsoleOp{
				e2e.ConsoleSendLine("0"),
			},
		},
		{
			name:       "export private binary",
			exportPath: c.privateExportPath,
			armor:      false,
			secret:     true,
			consoleOps: []e2e.SingularityConsoleOp{
				e2e.ConsoleSendLine("0"),
				e2e.ConsoleSendLine("e2etests"),
			},
		},
		{
			name:       "export public ascii",
			exportPath: c.publicExportASCIIPath,
			armor:      true,
			secret:     false,
			consoleOps: []e2e.SingularityConsoleOp{
				e2e.ConsoleSendLine("0"),
			},
		},
		{
			name:       "export private ascii",
			exportPath: c.privateExportASCIIPath,
			armor:      true,
			secret:     true,
			consoleOps: []e2e.SingularityConsoleOp{
				e2e.ConsoleSendLine("0"),
				e2e.ConsoleSendLine("e2etests"),
			},
		},
	}

	prepCmd := func(exportPath string, secret, armor bool) []string {
		cmd := []string{"export"}

		if armor {
			cmd = append(cmd, "--armor")
		}
		if secret {
			cmd = append(cmd, "--secret")
		}

		cmd = append(cmd, exportPath)

		return cmd
	}

	for _, tt := range tests {
		c.env.RunSingularity(
			t,
			e2e.WithPrivileges(false),
			e2e.WithCommand("key"),
			e2e.WithArgs(prepCmd(tt.exportPath, tt.secret, tt.armor)...),
			e2e.ConsoleRun(tt.consoleOps...),
			e2e.ExpectExit(0),
		)
	}
}

// singularityKeyImport will export a private, and public (binary and ASCII) key.
func (c *ctx) singularityKeyImport(t *testing.T) {
	tests := []struct {
		name       string
		exportPath string
		consoleOps []e2e.SingularityConsoleOp
	}{
		{
			name:       "import public binary",
			exportPath: c.publicExportPath,
			consoleOps: []e2e.SingularityConsoleOp{},
		},
		{
			name:       "import private binary",
			exportPath: c.privateExportPath,
			consoleOps: []e2e.SingularityConsoleOp{
				e2e.ConsoleSendLine("e2etests"),
				e2e.ConsoleSendLine("e2etests"),
				e2e.ConsoleSendLine("e2etests"),
			},
		},
		{
			name:       "import public ascii",
			exportPath: c.publicExportASCIIPath,
			consoleOps: []e2e.SingularityConsoleOp{},
		},
		{
			name:       "import private ascii",
			exportPath: c.privateExportASCIIPath,
			consoleOps: []e2e.SingularityConsoleOp{
				e2e.ConsoleSendLine("e2etests"),
				e2e.ConsoleSendLine("e2etests"),
				e2e.ConsoleSendLine("e2etests"),
			},
		},
	}

	prepCmd := func(exportPath string) []string {
		return []string{"import", exportPath}
	}

	for _, tt := range tests {
		c.singularityResetKeyring(t) // Remove the tmp keyring before each import
		c.env.RunSingularity(
			t,
			e2e.WithPrivileges(false),
			e2e.WithCommand("key"),
			e2e.WithArgs(prepCmd(tt.exportPath)...),
			e2e.ConsoleRun(tt.consoleOps...),
			e2e.ExpectExit(0),
		)
	}
	//t.Run("singularityKeyExport", c.singularityKeyExport)
}

func (c *ctx) singularityResetKeyring(t *testing.T) {
	// TODO: run this as non-root
	if err := os.RemoveAll(c.keyRing); err != nil {
		t.Fatalf("unable to remove tmp keyring directory: %s", err)
	}
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
		//t.Run("singularityKeyNewpairAndExport", c.singularityKeyNewpair)
		//t.Run("singularityKeyImportAndExport", c.singularityKeyImport)
		t.Run("singularityKeyNewpair", c.singularityKeyNewpair) // Generate a newpair (required for the other tests)
		t.Run("singularityKeyExport", c.singularityKeyExport)   // Export the newpair
		t.Run("singularityKeyImport", c.singularityKeyImport)   // Import the newpair (this will delete the old, tmp keyring before importing)
		t.Run("singularityKeyExport", c.singularityKeyExport)   // Re-export them, again (this will catch any issues if Singularity cant import correctly)
		t.Run("singularityKeyImport", c.singularityKeyImport)   // Finally, Re-import them
		t.Run("singularityKeyList", c.singularityKeyList)       // Then run 'key list' just because
	}
}
