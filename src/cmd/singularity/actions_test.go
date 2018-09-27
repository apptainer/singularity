// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"

	"github.com/sylabs/singularity/src/pkg/test"
)

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

// TestSingularityRun tests min fuctionality for singularity run
func TestSingularityRun(t *testing.T) {
	test.EnsurePrivilege(t)
	//build base image for tests
	imagePath := "./container.img"
	opts := buildOpts{
		sandbox:  false,
		writable: false,
	}
	if b, err := imageBuild(opts, imagePath, "../../../examples/busybox/Singularity"); err != nil {
		t.Log(string(b))
		t.Fatalf("unexpected failure: %v", err)
	}
	defer os.Remove(imagePath)

	tests := []struct {
		name string
		argv []string
		exit int
	}{
		{"NoCommand", []string{"run", imagePath}, 0},
		{"true", []string{"run", imagePath, "true"}, 0},
		{"false", []string{"run", imagePath, "false"}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			cmd := exec.Command(cmdPath, tt.argv...)
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
