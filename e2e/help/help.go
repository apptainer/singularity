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
	{[]string{"oci"}},
	{[]string{"oci", "attach"}},
	{[]string{"oci", "create"}},
	{[]string{"oci", "delete"}},
	{[]string{"oci", "exec"}},
	{[]string{"oci", "kill"}},
	{[]string{"oci", "mount"}},
	{[]string{"oci", "pause"}},
	{[]string{"oci", "resume"}},
	{[]string{"oci", "run"}},
	{[]string{"oci", "start"}},
	{[]string{"oci", "state"}},
	{[]string{"oci", "umount"}},
	{[]string{"oci", "update"}},
}

func TestHelp(t *testing.T) {
	c := test.NewCmd("singularity")

	for _, tc := range helpTests {
		name := fmt.Sprintf("%s.txt", strings.Join(tc.cmds, "-"))
		path := filepath.Join("help", name)
		args := append([]string{"help"}, tc.cmds...)

		t.Run(name, func(t *testing.T) {
			got := c.Run(t, args...).Stdout()

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
	t.Log(testenv)

	t.Run("TestHelp", TestHelp)
}
