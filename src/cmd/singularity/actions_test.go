// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
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

func imageExec(opts execOpts, imagePath string, command []string) ([]byte, error) {
	argv := []string{"exec"}
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

	return exec.Command(cmdPath, argv...).CombinedOutput()
}

// testSingularityRun tests min fuctionality for singularity run
func testSingularityRun(t *testing.T) {
	tests := []struct {
		binName string
		name    string
		argv    []string
		exit    int
	}{
		{cmdPath, "NoCommand", []string{"run", imagePath}, 0},
		{cmdPath, "true", []string{"run", imagePath, "true"}, 0},
		{cmdPath, "false", []string{"run", imagePath, "false"}, 1},
		// Testing run command properly hands arguments
		{"sh", "trueSTDIN", []string{"-c", fmt.Sprintf("singularity run %s foo | grep foo", imagePath)}, 0},
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

// testSingularityExec tests min fuctionality for singularity exec
func testSingularityExec(t *testing.T) {
	tests := []struct {
		binName string
		name    string
		argv    []string
		exit    int
	}{
		{cmdPath, "NoCommand", []string{"exec", imagePath}, 1},
		{cmdPath, "true", []string{"exec", imagePath, "true"}, 0},
		{cmdPath, "trueAbsPath", []string{"exec", imagePath, "/bin/true"}, 0},
		{cmdPath, "false", []string{"exec", imagePath, "false"}, 1},
		{cmdPath, "false", []string{"exec", imagePath, "/bin/false"}, 1},
		{"sh", "trueSTDIN", []string{"-c", fmt.Sprintf("echo hi | singularity exec %s grep hi", imagePath)}, 0},
		{"sh", "falseSTDIN", []string{"-c", fmt.Sprintf("echo bye | singularity exec %s grep hi", imagePath)}, 1},
		// Checking permissions
		{"sh", "trueSTDIN", []string{"-c", fmt.Sprintf("singularity exec %s id -u | grep `id -u`", imagePath)}, 0},
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

// testSingularityShell tests min fuctionality for singularity shell
func testSingularityShell(t *testing.T) {
	tests := []struct {
		binName string
		name    string
		argv    []string
		exit    int
	}{
		{cmdPath, "true", []string{"shell", imagePath, "-c", "true"}, 0},
		{"sh", "trueSTDIN", []string{"-c", fmt.Sprintf("echo true | singularity shell %s", imagePath)}, 0},
		{cmdPath, "false", []string{"shell", imagePath, "-c", "false"}, 1},
		{"sh", "falseSTDIN", []string{"-c", fmt.Sprintf("echo false | singularity shell %s", imagePath)}, 1},
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

func TestSingularityActions(t *testing.T) {
	test.EnsurePrivilege(t)
	opts := buildOpts{
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
	// singularity shell
	t.Run("shell", testSingularityShell)
}
