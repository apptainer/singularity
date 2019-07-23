// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package key

import (
	"fmt"
	"os"
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
)

type ctx struct {
	env e2e.TestEnv
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

func (c *ctx) singularityKeyRemovePub(t *testing.T) {

	fmt.Println("EENNNVVVV: ", os.Getenv("SINGULARITY_SYPGPDIR"))

}

// RunE2ETests is the main func to trigger the test suite
func RunE2ETests(env e2e.TestEnv) func(*testing.T) {
	c := &ctx{
		env: env,
	}

	return func(t *testing.T) {
		t.Run("singularityKeyList", c.singularityKeyList)
		t.Run("singularityKeyNewpair", c.singularityKeyNewpair)
		t.Run("singularityKeyList", c.singularityKeyList)
		t.Run("singularityKeyRemovePub", c.singularityKeyRemovePub)
		//t.Run("singularityKeyImport", c.singularityKey)
		//t.Run("singularityKeyExport", c.singularityKey)
	}
}
