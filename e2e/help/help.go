// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// This test sets singularity image specific environment variables and
// verifies that they are properly set.

package help

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/internal/pkg/test"
	"gotest.tools/assert"
	"gotest.tools/golden"
)

type testingEnv struct {
	// base env for running tests
	CmdPath     string `split_words:"true"`
	RunDisabled bool   `default:"false"`
}

var testenv testingEnv

var helpContentTests = []struct {
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

func testHelpContent(t *testing.T) {
	c := test.NewCmd(testenv.CmdPath)

	for _, tc := range helpContentTests {
		name := fmt.Sprintf("%s.txt", strings.Join(tc.cmds, "-"))
		path := filepath.Join("help", name)

		t.Run(name, func(t *testing.T) {
			got := c.Run(t, tc.cmds...).Stdout()

			assert.Assert(t, golden.String(got, path))
		})
	}
}

func testCommands(t *testing.T) {
	tests := []struct {
		name string
		argv []string
	}{
		{"Apps", []string{"apps"}},
		{"Bootstrap", []string{"bootstrap"}},
		{"Build", []string{"build"}},
		{"Check", []string{"check"}},
		{"Create", []string{"create"}},
		{"Exec", []string{"exec"}},
		{"Inspect", []string{"inspect"}},
		{"Mount", []string{"mount"}},
		{"Pull", []string{"pull"}},
		{"Run", []string{"run"}},
		{"Shell", []string{"shell"}},
		{"Test", []string{"test"}},
		{"InstanceDotStart", []string{"instance.start"}},
		{"InstanceDotList", []string{"instance.list"}},
		{"InstanceDotStop", []string{"instance.stop"}},
		{"InstanceStart", []string{"instance", "start"}},
		{"InstanceList", []string{"instance", "list"}},
		{"InstanceStop", []string{"instance", "stop"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			tests := []struct {
				name string
				argv []string
				skip bool
			}{
				{"PostFlagShort", append(tt.argv, "-h"), true}, // TODO
				{"PostFlagLong", append(tt.argv, "--help"), false},
				{"PostCommand", append(tt.argv, "help"), false},
				{"PreFlagShort", append([]string{"-h"}, tt.argv...), false},
				{"PreFlagLong", append([]string{"--help"}, tt.argv...), false},
				{"PreCommand", append([]string{"help"}, tt.argv...), false},
			}
			for _, tt := range tests {
				if tt.skip && !testenv.RunDisabled {
					t.Skip("disabled until issue addressed")
				}

				t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
					cmd := exec.Command(testenv.CmdPath, tt.argv...)
					if b, err := cmd.CombinedOutput(); err != nil {
						t.Log(string(b))
						t.Fatalf("unexpected failure running '%s': %s", strings.Join(tt.argv, " "), err)
					}
				}))
			}
		}))
	}

}

func testFailure(t *testing.T) {
	if !testenv.RunDisabled {
		t.Skip("disabled until issue addressed") // TODO
	}

	tests := []struct {
		name string
		argv []string
	}{
		{"HelpBogus", []string{"help", "bogus"}},
		{"BogusHelp", []string{"bogus", "help"}},
		{"HelpInstanceBogus", []string{"help", "instance", "bogus"}},
		{"ImageBogusHelp", []string{"image", "bogus", "help"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			cmd := exec.Command(testenv.CmdPath, tt.argv...)
			if b, err := cmd.CombinedOutput(); err == nil {
				t.Log(string(b))
				t.Fatalf("unexpected success running '%s'", strings.Join(tt.argv, " "))
			}
		}))
	}

}

func testSingularity(t *testing.T) {
	tests := []struct {
		name       string
		argv       []string
		shouldPass bool
	}{
		{"NoCommand", []string{}, false},
		{"FlagShort", []string{"-h"}, true},
		{"FlagLong", []string{"--help"}, true},
		{"Command", []string{"help"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			cmd := exec.Command(testenv.CmdPath, tt.argv...)
			b, err := cmd.CombinedOutput()
			if err != nil && tt.shouldPass {
				t.Log(string(b))
				t.Fatalf("unexpected failure running '%s': %s", strings.Join(tt.argv, " "), err)
			} else if err == nil && !tt.shouldPass {
				t.Log(string(b))
				t.Fatalf("unexpected success running '%s'", strings.Join(tt.argv, " "))
			}
		}))
	}

}

// RunE2ETests is the main func to trigger the test suite
func RunE2ETests(t *testing.T) {
	e2e.LoadEnv(t, &testenv)

	t.Log(testenv)

	// try to build from a non existen path
	t.Run("testCommands", testCommands)
	t.Run("testFailure", testFailure)
	t.Run("testSingularity", testSingularity)
	t.Run("testHelpContent", testHelpContent)
}
