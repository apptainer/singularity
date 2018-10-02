// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"

	"github.com/sylabs/singularity/src/pkg/test"
)

//build base image for tests
const imagePath = "./container.img"

type execOpts struct {
	binds   []string
	contain bool
	home    string
	workdir string
	pwd     string
}

// imageExec can be used to run/exec/shell a Singularity image
// it return the exitCode and err of the execution
func imageExec(t *testing.T, action string, opts execOpts, imagePath string, command []string) (stdout string, stderr string, exitCode int, err error) {
	// action can be run/exec/shell
	argv := []string{action}
	for _, bind := range opts.binds {
		argv = append(argv, "--bind", bind)
	}
	if opts.contain {
		argv = append(argv, "--contain")
	}
	if opts.home != "" {
		argv = append(argv, "--home", opts.home)
	}
	if opts.workdir != "" {
		argv = append(argv, "--workdir", opts.workdir)
	}
	if opts.pwd != "" {
		argv = append(argv, "--pwd", opts.pwd)
	}
	argv = append(argv, imagePath)
	argv = append(argv, command...)

	var outbuf, errbuf bytes.Buffer
	cmd := exec.Command(cmdPath, argv...)
	if err := cmd.Start(); err != nil {
		t.Fatalf("cmd.Start: %v", err)
	}

	err = cmd.Run()
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf

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

// testSingularityRun tests min fuctionality for singularity run
func testSingularityRun(t *testing.T) {
	tests := []struct {
		name   string
		image  string
		action string
		argv   []string
		execOpts
		exit          int
		expectSuccess bool
	}{
		{"NoCommand", imagePath, "run", []string{}, execOpts{}, 0, true},
		{"true", imagePath, "run", []string{"true"}, execOpts{}, 0, true},
		{"false", imagePath, "run", []string{"false"}, execOpts{}, 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			_, stderr, exitCode, err := imageExec(t, tt.action, tt.execOpts, tt.image, tt.argv)
			if tt.expectSuccess && (exitCode != 0) {
				t.Log(stderr)
				t.Fatalf("unexpected failure running '%v': %v", strings.Join(tt.argv, " "), err)
			} else if !tt.expectSuccess && (exitCode != 1) {
				t.Log(stderr)
				t.Fatalf("unexpected success running '%v'", strings.Join(tt.argv, " "))
			}
		}))
	}
}

// testSingularityExec tests min fuctionality for singularity exec
func testSingularityExec(t *testing.T) {
	tests := []struct {
		name   string
		image  string
		action string
		argv   []string
		execOpts
		exit          int
		expectSuccess bool
	}{
		{"NoCommand", imagePath, "exec", []string{}, execOpts{}, 1, false},
		{"true", imagePath, "exec", []string{"true"}, execOpts{}, 0, true},
		{"trueAbsPAth", imagePath, "exec", []string{"/bin/true"}, execOpts{}, 0, true},
		{"false", imagePath, "exec", []string{"false"}, execOpts{}, 1, false},
		{"falseAbsPath", imagePath, "exec", []string{"/bin/false"}, execOpts{}, 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			_, stderr, exitCode, err := imageExec(t, tt.action, tt.execOpts, tt.image, tt.argv)
			if tt.expectSuccess && (exitCode != 0) {
				t.Log(stderr)
				t.Fatalf("unexpected failure running '%v': %v", strings.Join(tt.argv, " "), err)
			} else if !tt.expectSuccess && (exitCode != 1) {
				t.Log(stderr)
				t.Fatalf("unexpected success running '%v'", strings.Join(tt.argv, " "))
			}
		}))
	}
}

// testSTDINPipe tests pipe stdin to singularity actions cmd
func testSTDINPipe(t *testing.T) {
	tests := []struct {
		binName string
		name    string
		argv    []string
		exit    int
	}{
		{"sh", "trueSTDIN", []string{"-c", fmt.Sprintf("echo hi | singularity exec %s grep hi", imagePath)}, 0},
		{"sh", "falseSTDIN", []string{"-c", fmt.Sprintf("echo bye | singularity exec %s grep hi", imagePath)}, 1},
		// Checking permissions
		{"sh", "permissions", []string{"-c", fmt.Sprintf("singularity exec %s id -u | grep `id -u`", imagePath)}, 0},
		// testing run command properly hands arguments
		{"sh", "arguments", []string{"-c", fmt.Sprintf("singularity run %s foo | grep foo", imagePath)}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			cmd := exec.Command(tt.binName, tt.argv...)
			if err := cmd.Start(); err != nil {
				t.Fatalf("cmd.Start: %v", err)
			}

			if err := cmd.Wait(); err != nil {
				exiterr, _ := err.(*exec.ExitError)
				status, _ := exiterr.Sys().(syscall.WaitStatus)
				if status.ExitStatus() != tt.exit {
					// The program has exited with an unexpected exit code
					{
						t.Fatalf("unexpected exit code '%v': for cmd %v", status.ExitStatus(), strings.Join(tt.argv, " "))
					}
				}
			}
		}))
	}
}

// TestRunFromURI tests min fuctionality for singularity run/exec URI://
func TestRunFromURI(t *testing.T) {
	tests := []struct {
		name   string
		image  string
		action string
		argv   []string
		execOpts
		exit          int
		expectSuccess bool
	}{
		// Run from supported URI's and check the runscript call works
		{"RunFromDocker", "docker://godlovedc/lolcow", "run", []string{}, execOpts{}, 0, true},
		{"RunFromLibrary", "library://sylabsed/examples/lolcow:latest", "run", []string{}, execOpts{}, 0, true},
		{"RunFromShub", "shub://GodloveD/lolcow", "run", []string{}, execOpts{}, 0, true},
		// exec from a supported URI's and check the exit code
		{"true", "docker://busybox:latest", "exec", []string{"true"}, execOpts{}, 0, true},
		{"true", "library://busybox:latest", "exec", []string{"true"}, execOpts{}, 0, true},
		{"true", "shub://singularityhub/busybox", "exec", []string{"true"}, execOpts{}, 0, true},
		{"false", "docker://busybox:latest", "exec", []string{"false"}, execOpts{}, 1, false},
		{"false", "library://busybox:latest", "exec", []string{"false"}, execOpts{}, 1, false},
		{"false", "shub://singularityhub/busybox", "exec", []string{"false"}, execOpts{}, 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			_, stderr, exitCode, err := imageExec(t, tt.action, tt.execOpts, tt.image, tt.argv)
			if tt.expectSuccess && (exitCode != 0) {
				t.Log(stderr)
				t.Fatalf("unexpected failure running '%v': %v", strings.Join(tt.argv, " "), err)
			} else if !tt.expectSuccess && (exitCode != 1) {
				t.Log(stderr)
				t.Fatalf("unexpected success running '%v'", strings.Join(tt.argv, " "))
			}
		}))
	}
}

func TestSingularityActions(t *testing.T) {
	test.EnsurePrivilege(t)
	opts := buildOpts{
		force:    true,
		sandbox:  false,
		writable: false,
	}
	if b, err := imageBuild(opts, imagePath, "../../../examples/busybox/Singularity"); err != nil {
		t.Log(string(b))
		t.Fatalf("unexpected failure: %v", err)
	}
	defer os.Remove(imagePath)

	// singularity run
	t.Run("run", testSingularityRun)
	// singularity exec
	t.Run("exec", testSingularityExec)
	// stdin pipe
	t.Run("STDIN", testSTDINPipe)
}
