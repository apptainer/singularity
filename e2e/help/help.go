// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// This test sets singularity image specific environment variables and
// verifies that they are properly set.

package help

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
)

type ctx struct {
	env e2e.TestEnv
}

var helpOciContentTests = []struct {
	name string
	cmds []string
}{
	// singularity oci
	{"HelpOci", []string{"oci"}},
	{"HelpOciAttach", []string{"oci", "attach"}},
	{"HelpOciCreate", []string{"oci", "create"}},
	{"HelpOciDelete", []string{"oci", "delete"}},
	{"HelpOciExec", []string{"oci", "exec"}},
	{"HelpOciKill", []string{"oci", "kill"}},
	{"HelpOciMount", []string{"oci", "mount"}},
	{"HelpOciPause", []string{"oci", "pause"}},
	{"HelpOciResume", []string{"oci", "resume"}},
	{"HelpOciRun", []string{"oci", "run"}},
	{"HelpOciStart", []string{"oci", "start"}},
	{"HelpOciState", []string{"oci", "state"}},
	{"HelpOciUmount", []string{"oci", "umount"}},
	{"HelpOciUpdate", []string{"oci", "update"}},
}

func (c *ctx) testHelpOciContent(t *testing.T) {
	for _, tc := range helpOciContentTests {

		name := fmt.Sprintf("help-%s.txt", strings.Join(tc.cmds, "-"))

		testHelpOciContentFn := func(t *testing.T, r *e2e.SingularityCmdResult) {
			path := filepath.Join("help", name)
			got := string(r.Stdout)
			assert.Assert(t, golden.String(got, path))
		}

		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tc.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithCommand("help"),
			e2e.WithArgs(tc.cmds...),
			e2e.PostRun(func(t *testing.T) {
				if t.Failed() {
					t.Fatalf("Failed to run help command on test: %s", tc.name)
				}
			}),
			e2e.ExpectExit(0, testHelpOciContentFn))

	}
}

func (c *ctx) testCommands(t *testing.T) {
	testCommands := []struct {
		name string
		cmd  string
	}{
		{"Build", "build"},
		{"Cache", "cache"},
		{"Capability", "capability"},
		{"Exec", "exec"},
		{"Instance", "instance"},
		{"Key", "key"},
		{"OCI", "oci"},
		{"Plugin", "plugin"},
		{"Inspect", "inspect"},
		{"Pull", "pull"},
		{"Push", "push"},
		{"Run", "run"},
		{"Run-help", "run-help"},
		{"Remote", "remote"},
		{"Search", "search"},
		{"Shell", "shell"},
		{"SIF", "sif"},
		{"Sign", "sign"},
		{"Test", "test"},
		{"Verify", "verify"},
		{"InstanceStart", "instance start"},
		{"InstanceList", "instance list"},
		{"InstanceStop", "instance stop"},
	}

	for _, tt := range testCommands {

		testFlags := []struct {
			name string
			argv string
			skip bool
		}{
			{"PostFlagShort", "-h", true}, // TODO
			{"PostFlagLong", "--help", false},
			{"PostCommand", "help", false},
			{"PreFlagShort", "-h", false},
			{"PreFlagLong", "--help", false},
			{"PreCommand", "help", false},
		}

		for _, tf := range testFlags {

			var cmdRun, argRun string

			if tf.name == "PostCommand" || tf.name == "PreCommand" {
				cmdRun = tf.argv
				argRun = ""
			} else {
				cmdRun = tt.cmd
				argRun = tf.argv
			}

			c.env.RunSingularity(
				t,
				e2e.AsSubtest(tf.name),
				e2e.WithProfile(e2e.UserProfile),
				e2e.WithCommand(cmdRun),
				e2e.WithArgs(argRun),
				e2e.PostRun(func(t *testing.T) {
					if t.Failed() {
						t.Fatalf("Failed to run help flag while running command:\n%s\n", tt.name)
					}
				}),
				e2e.PreRun(func(t *testing.T) {
					if tf.skip && !c.env.RunDisabled {
						t.Skip("disabled until issue addressed")
					}
				}),
				e2e.ExpectExit(0))
		}

	}

}

func (c *ctx) testFailure(t *testing.T) {
	if !c.env.RunDisabled {
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

		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithArgs(tt.argv...),
			e2e.PostRun(func(t *testing.T) {
				if !t.Failed() {
					t.Fatalf("While running command:\n%s\nUnexpected success", tt.name)
				}
			}),
			e2e.ExpectExit(0))
	}

}

// Expected first lines of the help command output based on what options is used
const (
	helpExpectedOutput     = ""
	helpHelpExpectedOutput = "Help about any command"
)

func (c *ctx) testSingularity(t *testing.T) {
	tests := []struct {
		name           string
		argv           []string
		expectedOutput string // Part of the output that is expected
		exit           int
	}{
		{
			name:           "NoCommand",
			argv:           []string{},
			expectedOutput: helpExpectedOutput,
			exit:           1,
		},
		{
			name:           "FlagShort",
			argv:           []string{"-h"},
			expectedOutput: helpExpectedOutput,
			exit:           0,
		},
		{
			name:           "FlagLong",
			argv:           []string{"--help"},
			expectedOutput: helpExpectedOutput,
			exit:           0,
		},
		{
			name:           "Command",
			argv:           []string{"help"},
			expectedOutput: helpExpectedOutput,
			exit:           0,
		},
		{
			name:           "CommandAndLongFlag",
			argv:           []string{"help", "--help"},
			expectedOutput: helpHelpExpectedOutput,
			exit:           0,
		},
		{
			name:           "CommandAndShortFlag",
			argv:           []string{"help", "-h"},
			expectedOutput: helpHelpExpectedOutput,
			exit:           0,
		},
	}

	for _, tt := range tests {

		printSuccessOrFailureFn := func(t *testing.T, r *e2e.SingularityCmdResult) {
			if r.Stdout != nil {
				t.Logf(string(r.Stdout) + "\n")
			}
			if r.Stderr != nil {
				t.Logf(string(r.Stderr) + "\n")
			}
		}

		c.env.RunSingularity(t,
			e2e.AsSubtest(tt.name),
			e2e.WithProfile(e2e.UserProfile),
			e2e.WithArgs(tt.argv...),
			e2e.ExpectExit(tt.exit,
				printSuccessOrFailureFn,
				e2e.ExpectOutput(e2e.RegexMatch, `^`+tt.expectedOutput),
			),
		)
	}

}

// E2ETests is the main func to trigger the test suite
func E2ETests(env e2e.TestEnv) func(*testing.T) {
	c := &ctx{
		env: env,
	}

	return func(t *testing.T) {
		// try to build from a non existen path
		t.Run("testCommands", c.testCommands)
		t.Run("testFailure", c.testFailure)
		t.Run("testSingularity", c.testSingularity)
		t.Run("testHelpContent", c.testHelpOciContent)
	}
}
