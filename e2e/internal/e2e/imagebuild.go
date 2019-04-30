// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package e2e

import (
	"bytes"
	"io/ioutil"
	"log"
	"os/exec"
	"path"
	"strings"
	"testing"
	"text/template"

	"github.com/sylabs/singularity/internal/pkg/test"
)

// BuildOpts define image build options
type BuildOpts struct {
	Force   bool
	Sandbox bool
	Env     []string
}

type DefFileDetails struct {
	Bootstrap string
	From      string
	Registry  string
	Namespace string
	Labels    map[string]string
}

// ImageBuild builds an image based on the Opts
func ImageBuild(cmdPath string, opts BuildOpts, imagePath, buildSpec string) ([]byte, error) {
	var argv []string
	argv = append(argv, "build")
	if opts.Force {
		argv = append(argv, "--force")
	}
	if opts.Sandbox {
		argv = append(argv, "--sandbox")
	}
	argv = append(argv, imagePath, buildSpec)

	cmd := exec.Command(cmdPath, argv...)
	cmd.Env = opts.Env

	return cmd.CombinedOutput()
}

// ImageVerify checks for an image integrity
func ImageVerify(t *testing.T, cmdPath string, imagePath string, labels bool, runDisabled bool) {
	type testSpec struct {
		name          string
		execArgs      []string
		expectSuccess bool
	}
	tests := []testSpec{
		{"False", []string{"false"}, false},
		{"RunScript", []string{"test", "-f", "/.singularity.d/runscript"}, true},
		{"OneBase", []string{"test", "-f", "/.singularity.d/env/01-base.sh"}, true},
		{"ActionsShell", []string{"test", "-f", "/.singularity.d/actions/shell"}, true},
		{"ActionsExec", []string{"test", "-f", "/.singularity.d/actions/exec"}, true},
		{"ActionsRun", []string{"test", "-f", "/.singularity.d/actions/run"}, true},
		{"Environment", []string{"test", "-L", "/environment"}, true},
		{"Singularity", []string{"test", "-L", "/singularity"}, true},
	}
	if labels && runDisabled { // TODO
		tests = append(tests, testSpec{"Labels", []string{"test", "-f", "/.singularity.d/labels.json"}, true})
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			_, stderr, exitCode, err := ImageExec(t, cmdPath, "exec", ExecOpts{}, imagePath, tt.execArgs)
			if tt.expectSuccess && (exitCode != 0) {
				t.Log(stderr)
				t.Fatalf("unexpected failure running '%v': %v", strings.Join(tt.execArgs, " "), err)
			} else if !tt.expectSuccess && (exitCode != 1) {
				t.Log(stderr)
				t.Fatalf("unexpected success running '%v'", strings.Join(tt.execArgs, " "))
			}
		}))
	}
}

// PrepareDefFile reads a template from a file, applies data to it, writes the
// contents to disk, and returns the path.
func PrepareDefFile(dfd DefFileDetails) (outputPath string) {
	tmpl, err := template.ParseFiles(path.Join("testdata", "deffile.tmpl"))
	if err != nil {
		log.Fatalf("failed to parse template: %v", err)
	}

	f, err := ioutil.TempFile("", "TestTemplate-")
	if err != nil {
		log.Fatalf("failed to open temp file: %v", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, dfd); err != nil {
		log.Fatalf("failed to execute template: %v", err)
	}

	return f.Name()
}

// ExecOpts define options for singularity actions
type ExecOpts struct {
	Binds     []string
	Security  []string
	Overlay   []string
	DropCaps  string
	Home      string
	Workdir   string
	Pwd       string
	App       string
	KeepPrivs bool
	Contain   bool
	NoHome    bool
	Userns    bool
}

// ImageExec can be used to run/exec/shell a Singularity image
// it return the exitCode and err of the execution
func ImageExec(t *testing.T, cmdPath string, action string, opts ExecOpts, imagePath string, command []string) (stdout string, stderr string, exitCode int, err error) {
	// action can be run/exec/shell
	argv := []string{action}
	for _, bind := range opts.Binds {
		argv = append(argv, "--bind", bind)
	}
	for _, sec := range opts.Security {
		argv = append(argv, "--security", sec)
	}
	if opts.KeepPrivs {
		argv = append(argv, "--keep-privs")
	}
	if opts.DropCaps != "" {
		argv = append(argv, "--drop-caps", opts.DropCaps)
	}
	if opts.Contain {
		argv = append(argv, "--contain")
	}
	if opts.Userns {
		argv = append(argv, "--userns")
	}
	if opts.NoHome {
		argv = append(argv, "--no-home")
	}
	if opts.Home != "" {
		argv = append(argv, "--home", opts.Home)
	}
	for _, fs := range opts.Overlay {
		argv = append(argv, "--overlay", fs)
	}
	if opts.Workdir != "" {
		argv = append(argv, "--workdir", opts.Workdir)
	}
	if opts.Pwd != "" {
		argv = append(argv, "--pwd", opts.Pwd)
	}
	if opts.App != "" {
		argv = append(argv, "--app", opts.App)
	}
	argv = append(argv, imagePath)
	argv = append(argv, command...)

	var outbuf, errbuf bytes.Buffer
	cmd := exec.Command(cmdPath, argv...)

	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf

	if err := cmd.Start(); err != nil {
		t.Fatalf("cmd.Start: %v", err)
	}

	// retrieve exit code
	if err := cmd.Wait(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			// The program has exited with an exit code != 0
			exitCode = 1
		}
	}

	stdout = outbuf.String()
	stderr = errbuf.String()

	return
}

// GenericExec executes an external program and returns its stdout and stderr.
// If err != nil, the program did not execute successfully.
func GenericExec(cmdPath string, argv ...string) (stdout string, stderr string, err error) {
	var stdoutBuffer, stderrBuffer bytes.Buffer

	// Execute command
	cmd := exec.Command(cmdPath, argv...)
	cmd.Stdout = &stdoutBuffer
	cmd.Stderr = &stderrBuffer
	if err = cmd.Start(); err != nil {
		return
	}

	// Wait for command to finish and set stdout/stderr
	err = cmd.Wait()
	stdout = stdoutBuffer.String()
	stderr = stderrBuffer.String()
	return
}
