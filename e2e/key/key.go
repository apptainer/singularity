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
		fmt.Println("STDOUT: ", string(r.Stdout))
	}

	c.env.RunSingularity(
		t,
		e2e.WithPrivileges(false),
		e2e.WithCommand("key"),
		e2e.WithArgs("list"),
		//		e2e.PostRun(func(t *testing.T) {
		//			defer os.Remove(defFile)
		//			defer os.RemoveAll(imagePath)
		//
		//			e2e.DefinitionImageVerify(t, c.env.CmdPath, imagePath, tt.dfd)
		//		}),
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
			exportPath: c.publicExportPath,
			armor:      false,
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

func (c *ctx) singularityResetKeyring(t *testing.T) {
	fmt.Println("EENNNVVVV: ", os.Getenv("SINGULARITY_SYPGPDIR"))
	fmt.Println("Removing: ", c.keyRing)

	//if err := os.RemoveAll(c.keyRing); err != nil {
	//	t.Fatalf("unable to remove tmp keyring: %s", err)
	//}
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

	fmt.Println("NEW__EENNNVVVV: ", os.Getenv("SINGULARITY_SYPGPDIR"))

	return func(t *testing.T) {
		t.Run("singularityKeyList", c.singularityKeyList)
		t.Run("singularityKeyNewpair", c.singularityKeyNewpair)
		t.Run("singularityKeyList", c.singularityKeyList)
		t.Run("singularityKeyExport", c.singularityKeyExport)
		t.Run("singularityResetKeyring", c.singularityResetKeyring)
		//t.Run("singularityKeyList", c.singularityKeyList)
	}
}
