// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package help

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kelseyhightower/envconfig"
	"github.com/sylabs/singularity/internal/pkg/test"
	"gotest.tools/assert"
	"gotest.tools/golden"
)

type testingEnv struct {
	// base env for running tests
	CmdPath     string `split_words:"true"`
	TestDir     string `split_words:"true"`
	RunDisabled bool   `default:"false"`
}

var testenv testingEnv

var helpTests = []struct {
	cmds []string
}{
	// singularity oci
	{[]string{"help", "oci"}},
	{[]string{"help", "oci", "attach"}},
	{[]string{"help", "oci", "create"}},
	{[]string{"help", "oci", "delete"}},
	{[]string{"help", "oci", "exec"}},
	{[]string{"help", "oci", "kill"}},
	{[]string{"help", "oci", "mount"}},
	{[]string{"help", "oci", "pause"}},
	{[]string{"help", "oci", "resume"}},
	{[]string{"help", "oci", "run"}},
	{[]string{"help", "oci", "start"}},
	{[]string{"help", "oci", "state"}},
	{[]string{"help", "oci", "umount"}},
	{[]string{"help", "oci", "update"}},
}

func testHelp(t *testing.T) {
	c := test.NewCmd(testenv.CmdPath)

	for _, tc := range helpTests {
		name := fmt.Sprintf("%s.txt", strings.Join(tc.cmds, "-"))
		path := filepath.Join("help", name)

		t.Run(name, func(t *testing.T) {
			got := c.Run(t, tc.cmds...).Stdout()

			assert.Assert(t, golden.String(got, path))
		})
	}
}

//RunE2ETests is the main func to trigger the test suite
func RunE2ETests(t *testing.T) {
	err := envconfig.Process("E2E", &testenv)
	if err != nil {
		t.Fatal(err.Error())
	}

	t.Run("TestHelp", testHelp)
}
