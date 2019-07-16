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
	"gotest.tools/assert"
	"gotest.tools/golden"
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

		name := fmt.Sprintf("%s.txt", strings.Join(tc.cmds, "-"))

		testHelpOciContentFn := func(t *testing.T, r *e2e.SingularityCmdResult) {
			path := filepath.Join("help", name)
			got := string(r.Stdout)
			assert.Assert(t, golden.String(got, path))
		}

		e2e.RunSingularity(t, tc.name, e2e.WithCommand("help"), e2e.WithArgs(tc.cmds...),
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
		{"Check", "check"},
		{"Create", "create"},
		{"Exec", "exec"},
		{"Inspect", "inspect"},
		{"Mount", "mount"},
		{"Pull", "pull"},
		{"Run", "run"},
		{"Shell", "shell"},
		{"Test", "test"},
		{"InstanceDotStart", "instance.start"},
		{"InstanceDotList", "instance.list"},
		{"InstanceDotStop", "instance.stop"},
		{"InstanceStart", "instance start"},
		{"InstanceList", "instance list"},
		{"InstanceStop", "instance stop"},
	}

	for _, tt := range testCommands {

		testFlags := []struct {
			name string
			argv []string
			skip bool
		}{
			{"PostFlagShort", append([]string{tt.cmd}, "-h"), true}, // TODO
			{"PostFlagLong", append([]string{tt.cmd}, "--help"), false},
			{"PostCommand", append([]string{tt.cmd}, "help"), false},
			{"PreFlagShort", append([]string{tt.cmd}, "-h"), false},
			{"PreFlagLong", append([]string{tt.cmd}, "--help"), false},
			{"PreCommand", append([]string{tt.cmd}, "help"), false},
		}

		for _, tf := range testFlags {

			e2e.RunSingularity(t, tf.name, e2e.WithCommand(tt.cmd), e2e.WithArgs(tf.argv...),
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

		e2e.RunSingularity(t, tt.name, e2e.WithArgs(tt.argv...),
			e2e.PostRun(func(t *testing.T) {
				if !t.Failed() {
					t.Fatalf("While running command:\n%s\nUnexpected success", tt.name)
				}
			}),
			e2e.ExpectExit(0))
	}

}

func (c *ctx) testSingularity(t *testing.T) {
	tests := []struct {
		name string
		argv []string
		exit int
	}{
		{"NoCommand", []string{}, 0},
		{"FlagShort", []string{"-h"}, 1},
		{"FlagLong", []string{"--help"}, 1},
		{"Command", []string{"help"}, 0},
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

		e2e.RunSingularity(t, tt.name, e2e.WithArgs(tt.argv...),
			e2e.ExpectExit(tt.exit, printSuccessOrFailureFn))
	}

}

// RunE2ETests is the main func to trigger the test suite
func RunE2ETests(env e2e.TestEnv) func(*testing.T) {
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
